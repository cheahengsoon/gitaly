package smarthttp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/git/hooks"
	"gitlab.com/gitlab-org/gitaly/internal/git/pktline"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	gitalyhook "gitlab.com/gitlab-org/gitaly/internal/gitaly/hook"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/hook"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/transaction"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/helper/text"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	pconfig "gitlab.com/gitlab-org/gitaly/internal/praefect/config"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/metadata"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/streamio"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	uploadPackCapabilities = "report-status side-band-64k agent=git/2.12.0"
)

func TestSuccessfulReceivePackRequest(t *testing.T) {
	defer func(dir string) { config.Config.GitlabShell.Dir = dir }(config.Config.GitlabShell.Dir)
	config.Config.GitlabShell.Dir = "/foo/bar/gitlab-shell"

	hookOutputFile, cleanup := testhelper.CaptureHookEnv(t)
	defer cleanup()

	serverSocketPath, stop := runSmartHTTPServer(t, config.Config)
	defer stop()

	repo, repoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	client, conn := newSmartHTTPClient(t, serverSocketPath, config.Config.Auth.Token)
	defer conn.Close()

	ctx, cancel := testhelper.Context()
	defer cancel()

	stream, err := client.PostReceivePack(ctx)
	require.NoError(t, err)

	push := newTestPush(t, nil)

	projectPath := "project/path"

	repo.GlProjectPath = projectPath
	firstRequest := &gitalypb.PostReceivePackRequest{Repository: repo, GlUsername: "user", GlId: "123", GlRepository: "project-456"}
	response := doPush(t, stream, firstRequest, push.body)

	expectedResponse := "0049\x01000eunpack ok\n0019ok refs/heads/master\n0019ok refs/heads/branch\n00000000"
	require.Equal(t, expectedResponse, string(response), "Expected response to be %q, got %q", expectedResponse, response)

	// The fact that this command succeeds means that we got the commit correctly, no further checks should be needed.
	testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "show", push.newHead)

	envData, err := ioutil.ReadFile(hookOutputFile)
	require.NoError(t, err, "get git env data")

	payload, err := git.HooksPayloadFromEnv(strings.Split(string(envData), "\n"))
	require.NoError(t, err)

	// Compare the repository up front so that we can use require.Equal for
	// the remaining values.
	testhelper.ProtoEqual(t, repo, payload.Repo)
	payload.Repo = nil

	// If running tests with Praefect, then these would be set, but we have
	// no way of figuring out their actual contents. So let's just remove
	// that data, too.
	payload.Transaction = nil
	payload.Praefect = nil

	require.Equal(t, git.HooksPayload{
		BinDir:              config.Config.BinDir,
		GitPath:             config.Config.Git.BinPath,
		InternalSocket:      config.Config.GitalyInternalSocketPath(),
		InternalSocketToken: config.Config.Auth.Token,
		ReceiveHooksPayload: &git.ReceiveHooksPayload{
			UserID:   "123",
			Username: "user",
			Protocol: "http",
		},
		RequestedHooks: git.ReceivePackHooks,
	}, payload)
}

func TestSuccessfulReceivePackRequestWithGitProtocol(t *testing.T) {
	defer func(old config.Cfg) { config.Config = old }(config.Config)

	cfg, restore := testhelper.EnableGitProtocolV2Support(t, config.Config)
	defer restore()
	config.Config = cfg

	serverSocketPath, stop := runSmartHTTPServer(t, config.Config)
	defer stop()

	repo, repoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	client, conn := newSmartHTTPClient(t, serverSocketPath, config.Config.Auth.Token)
	defer conn.Close()

	ctx, cancel := testhelper.Context()
	defer cancel()

	stream, err := client.PostReceivePack(ctx)
	require.NoError(t, err)

	push := newTestPush(t, nil)
	firstRequest := &gitalypb.PostReceivePackRequest{Repository: repo, GlId: "user-123", GlRepository: "project-123", GitProtocol: git.ProtocolV2}
	doPush(t, stream, firstRequest, push.body)

	envData := testhelper.GetGitEnvData(t)
	require.Equal(t, fmt.Sprintf("GIT_PROTOCOL=%s\n", git.ProtocolV2), envData)

	// The fact that this command succeeds means that we got the commit correctly, no further checks should be needed.
	testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "show", push.newHead)
}

func TestFailedReceivePackRequestWithGitOpts(t *testing.T) {
	serverSocketPath, stop := runSmartHTTPServer(t, config.Config)
	defer stop()

	repo, _, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	client, conn := newSmartHTTPClient(t, serverSocketPath, config.Config.Auth.Token)
	defer conn.Close()

	ctx, cancel := testhelper.Context()
	defer cancel()

	stream, err := client.PostReceivePack(ctx)
	require.NoError(t, err)

	push := newTestPush(t, nil)
	firstRequest := &gitalypb.PostReceivePackRequest{Repository: repo, GlId: "user-123", GlRepository: "project-123", GitConfigOptions: []string{"receive.MaxInputSize=1"}}
	response := doPush(t, stream, firstRequest, push.body)

	expectedResponse := "002e\x02fatal: pack exceeds maximum allowed size\n0081\x010028unpack unpack-objects abnormal exit\n0028ng refs/heads/master unpacker error\n0028ng refs/heads/branch unpacker error\n00000000"
	require.Equal(t, expectedResponse, string(response), "Expected response to be %q, got %q", expectedResponse, response)
}

func TestFailedReceivePackRequestDueToHooksFailure(t *testing.T) {
	hookDir, cleanup := testhelper.TempDir(t)
	defer cleanup()

	defer func(override string) {
		hooks.Override = override
	}(hooks.Override)
	hooks.Override = hookDir

	require.NoError(t, os.MkdirAll(hooks.Path(config.Config), 0755))

	hookContent := []byte("#!/bin/sh\nexit 1")
	ioutil.WriteFile(filepath.Join(hooks.Path(config.Config), "pre-receive"), hookContent, 0755)

	serverSocketPath, stop := runSmartHTTPServer(t, config.Config)
	defer stop()

	repo, _, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	client, conn := newSmartHTTPClient(t, serverSocketPath, config.Config.Auth.Token)
	defer conn.Close()

	ctx, cancel := testhelper.Context()
	defer cancel()

	stream, err := client.PostReceivePack(ctx)
	require.NoError(t, err)

	push := newTestPush(t, nil)
	firstRequest := &gitalypb.PostReceivePackRequest{Repository: repo, GlId: "user-123", GlRepository: "project-123"}
	response := doPush(t, stream, firstRequest, push.body)

	expectedResponse := "007d\x01000eunpack ok\n0033ng refs/heads/master pre-receive hook declined\n0033ng refs/heads/branch pre-receive hook declined\n00000000"
	require.Equal(t, expectedResponse, string(response), "Expected response to be %q, got %q", expectedResponse, response)
}

func doPush(t *testing.T, stream gitalypb.SmartHTTPService_PostReceivePackClient, firstRequest *gitalypb.PostReceivePackRequest, body io.Reader) []byte {
	require.NoError(t, stream.Send(firstRequest))

	sw := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&gitalypb.PostReceivePackRequest{Data: p})
	})
	_, err := io.Copy(sw, body)
	require.NoError(t, err)

	require.NoError(t, stream.CloseSend())

	responseBuffer := bytes.Buffer{}
	rr := streamio.NewReader(func() ([]byte, error) {
		resp, err := stream.Recv()
		return resp.GetData(), err
	})
	_, err = io.Copy(&responseBuffer, rr)
	require.NoError(t, err)

	return responseBuffer.Bytes()
}

type pushData struct {
	newHead string
	body    io.Reader
}

func newTestPush(t *testing.T, fileContents []byte) *pushData {
	_, repoPath, localCleanup := testhelper.NewTestRepoWithWorktree(t)
	defer localCleanup()

	oldHead, newHead := createCommit(t, repoPath, fileContents)

	// ReceivePack request is a packet line followed by a packet flush, then the pack file of the objects we want to push.
	// This is explained a bit in https://git-scm.com/book/en/v2/Git-Internals-Transfer-Protocols#_uploading_data
	// We form the packet line the same way git executable does: https://github.com/git/git/blob/d1a13d3fcb252631361a961cb5e2bf10ed467cba/send-pack.c#L524-L527
	requestBuffer := &bytes.Buffer{}

	pkt := fmt.Sprintf("%s %s refs/heads/master\x00 %s", oldHead, newHead, uploadPackCapabilities)
	fmt.Fprintf(requestBuffer, "%04x%s", len(pkt)+4, pkt)

	pkt = fmt.Sprintf("%s %s refs/heads/branch", git.ZeroOID, newHead)
	fmt.Fprintf(requestBuffer, "%04x%s", len(pkt)+4, pkt)

	fmt.Fprintf(requestBuffer, "%s", pktFlushStr)

	// We need to get a pack file containing the objects we want to push, so we use git pack-objects
	// which expects a list of revisions passed through standard input. The list format means
	// pack the objects needed if I have oldHead but not newHead (think of it from the perspective of the remote repo).
	// For more info, check the man pages of both `git-pack-objects` and `git-rev-list --objects`.
	stdin := strings.NewReader(fmt.Sprintf("^%s\n%s\n", oldHead, newHead))

	// The options passed are the same ones used when doing an actual push.
	pack := testhelper.MustRunCommand(t, stdin, "git", "-C", repoPath, "pack-objects", "--stdout", "--revs", "--thin", "--delta-base-offset", "-q")
	requestBuffer.Write(pack)

	return &pushData{newHead: newHead, body: requestBuffer}
}

// createCommit creates a commit on HEAD with a file containing the
// specified contents.
func createCommit(t *testing.T, repoPath string, fileContents []byte) (oldHead string, newHead string) {
	commitMsg := fmt.Sprintf("Testing ReceivePack RPC around %d", time.Now().Unix())
	committerName := "Scrooge McDuck"
	committerEmail := "scrooge@mcduck.com"

	// The latest commit ID on the remote repo
	oldHead = text.ChompBytes(testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "rev-parse", "master"))

	changedFile := "README.md"
	require.NoError(t, ioutil.WriteFile(filepath.Join(repoPath, changedFile), fileContents, 0644))

	testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "add", changedFile)
	testhelper.MustRunCommand(t, nil, "git", "-C", repoPath,
		"-c", fmt.Sprintf("user.name=%s", committerName),
		"-c", fmt.Sprintf("user.email=%s", committerEmail),
		"commit", "-m", commitMsg)

	// The commit ID we want to push to the remote repo
	newHead = text.ChompBytes(testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "rev-parse", "master"))

	return oldHead, newHead
}

func TestFailedReceivePackRequestDueToValidationError(t *testing.T) {
	serverSocketPath, stop := runSmartHTTPServer(t, config.Config)
	defer stop()

	client, conn := newSmartHTTPClient(t, serverSocketPath, config.Config.Auth.Token)
	defer conn.Close()

	rpcRequests := []gitalypb.PostReceivePackRequest{
		{Repository: &gitalypb.Repository{StorageName: "fake", RelativePath: "path"}, GlId: "user-123"}, // Repository doesn't exist
		{Repository: nil, GlId: "user-123"}, // Repository is nil
		{Repository: &gitalypb.Repository{StorageName: "default", RelativePath: "path/to/repo"}, GlId: ""},                               // Empty GlId
		{Repository: &gitalypb.Repository{StorageName: "default", RelativePath: "path/to/repo"}, GlId: "user-123", Data: []byte("Fail")}, // Data exists on first request
	}

	for _, rpcRequest := range rpcRequests {
		t.Run(fmt.Sprintf("%v", rpcRequest), func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()
			stream, err := client.PostReceivePack(ctx)
			require.NoError(t, err)

			require.NoError(t, stream.Send(&rpcRequest))
			stream.CloseSend()

			err = drainPostReceivePackResponse(stream)
			testhelper.RequireGrpcError(t, err, codes.InvalidArgument)
		})
	}
}

func TestInvalidTimezone(t *testing.T) {
	_, localRepoPath, localCleanup := testhelper.NewTestRepoWithWorktree(t)
	defer localCleanup()

	head := text.ChompBytes(testhelper.MustRunCommand(t, nil, "git", "-C", localRepoPath, "rev-parse", "HEAD"))
	tree := text.ChompBytes(testhelper.MustRunCommand(t, nil, "git", "-C", localRepoPath, "rev-parse", "HEAD^{tree}"))

	buf := new(bytes.Buffer)
	buf.WriteString("tree " + tree + "\n")
	buf.WriteString("parent " + head + "\n")
	buf.WriteString("author Au Thor <author@example.com> 1313584730 +051800\n")
	buf.WriteString("committer Au Thor <author@example.com> 1313584730 +051800\n")
	buf.WriteString("\n")
	buf.WriteString("Commit message\n")
	commit := text.ChompBytes(testhelper.MustRunCommand(t, buf, "git", "-C", localRepoPath, "hash-object", "-t", "commit", "--stdin", "-w"))

	stdin := strings.NewReader(fmt.Sprintf("^%s\n%s\n", head, commit))
	pack := testhelper.MustRunCommand(t, stdin, "git", "-C", localRepoPath, "pack-objects", "--stdout", "--revs", "--thin", "--delta-base-offset", "-q")

	pkt := fmt.Sprintf("%s %s refs/heads/master\x00 %s", head, commit, "report-status side-band-64k agent=git/2.12.0")
	body := &bytes.Buffer{}
	fmt.Fprintf(body, "%04x%s%s", len(pkt)+4, pkt, pktFlushStr)
	body.Write(pack)

	_, cleanup := testhelper.CaptureHookEnv(t)
	defer cleanup()

	socket, stop := runSmartHTTPServer(t, config.Config)
	defer stop()

	repo, repoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	client, conn := newSmartHTTPClient(t, socket, config.Config.Auth.Token)
	defer conn.Close()

	ctx, cancel := testhelper.Context()
	defer cancel()

	stream, err := client.PostReceivePack(ctx)
	require.NoError(t, err)
	firstRequest := &gitalypb.PostReceivePackRequest{
		Repository:       repo,
		GlId:             "user-123",
		GlRepository:     "project-456",
		GitConfigOptions: []string{"receive.fsckObjects=true"},
	}
	response := doPush(t, stream, firstRequest, body)

	expectedResponse := "0030\x01000eunpack ok\n0019ok refs/heads/master\n00000000"
	require.Equal(t, expectedResponse, string(response), "Expected response to be %q, got %q", expectedResponse, response)
	testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "show", commit)
}

func drainPostReceivePackResponse(stream gitalypb.SmartHTTPService_PostReceivePackClient) error {
	var err error
	for err == nil {
		_, err = stream.Recv()
	}
	return err
}

func TestPostReceivePackToHooks(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	secretToken := "secret token"
	glRepository := "some_repo"
	glID := "key-123"

	config.Config.Auth.Token = "abc123"

	server, socket := runSmartHTTPHookServiceServer(t, config.Config)
	defer server.Stop()

	client, conn := newSmartHTTPClient(t, "unix://"+socket, config.Config.Auth.Token)
	defer conn.Close()

	tempGitlabShellDir, cleanup := testhelper.TempDir(t)
	defer cleanup()

	defer func(cfg config.Cfg) {
		config.Config = cfg
	}(config.Config)
	config.Config.GitlabShell.Dir = tempGitlabShellDir

	repo, testRepoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	push := newTestPush(t, nil)
	oldHead := text.ChompBytes(testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "rev-parse", "HEAD"))

	changes := fmt.Sprintf("%s %s refs/heads/master\n", oldHead, push.newHead)

	serverURL, cleanup := testhelper.NewGitlabTestServer(t, testhelper.GitlabTestServerOptions{
		User:                        "",
		Password:                    "",
		SecretToken:                 secretToken,
		GLID:                        glID,
		GLRepository:                glRepository,
		Changes:                     changes,
		PostReceiveCounterDecreased: true,
		Protocol:                    "http",
	})
	defer cleanup()

	testhelper.WriteShellSecretFile(t, tempGitlabShellDir, secretToken)

	cleanup = testhelper.WriteCheckNewObjectExistsHook(t, config.Config.Git.BinPath, testRepoPath)
	defer cleanup()

	config.Config.Gitlab.URL = serverURL
	config.Config.Gitlab.SecretFile = filepath.Join(tempGitlabShellDir, ".gitlab_shell_secret")

	stream, err := client.PostReceivePack(ctx)
	require.NoError(t, err)

	firstRequest := &gitalypb.PostReceivePackRequest{
		Repository:   repo,
		GlId:         glID,
		GlRepository: glRepository,
	}

	response := doPush(t, stream, firstRequest, push.body)

	expectedResponse := "0049\x01000eunpack ok\n0019ok refs/heads/master\n0019ok refs/heads/branch\n00000000"
	require.Equal(t, expectedResponse, string(response), "Expected response to be %q, got %q", expectedResponse, response)
	require.Equal(t, io.EOF, drainPostReceivePackResponse(stream))
}

func runSmartHTTPHookServiceServer(t *testing.T, cfg config.Cfg) (*grpc.Server, string) {
	server := testhelper.NewTestGrpcServer(t, nil, nil)

	serverSocketPath := testhelper.GetTemporaryGitalySocketFileName(t)
	listener, err := net.Listen("unix", serverSocketPath)
	if err != nil {
		t.Fatal(err)
	}
	internalListener, err := net.Listen("unix", cfg.GitalyInternalSocketPath())
	if err != nil {
		t.Fatal(err)
	}

	locator := config.NewLocator(cfg)
	txManager := transaction.NewManager(cfg)
	hookManager := gitalyhook.NewManager(locator, txManager, gitalyhook.GitlabAPIStub, cfg)
	gitCmdFactory := git.NewExecCommandFactory(cfg)

	gitalypb.RegisterSmartHTTPServiceServer(server, NewServer(cfg, locator, gitCmdFactory))
	gitalypb.RegisterHookServiceServer(server, hook.NewServer(cfg, hookManager, gitCmdFactory))
	reflection.Register(server)

	go server.Serve(listener)
	go server.Serve(internalListener)

	return server, "unix://" + serverSocketPath
}

func TestPostReceiveWithTransactionsViaPraefect(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	defer func(cfg config.Cfg) {
		config.Config = cfg
	}(config.Config)

	secretToken := "secret token"
	glID := "key-1234"
	glRepository := "some_repo"
	gitlabUser := "gitlab_user-1234"
	gitlabPassword := "gitlabsecret9887"

	repo, repoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	opts := testhelper.GitlabTestServerOptions{
		User:         gitlabUser,
		Password:     gitlabPassword,
		SecretToken:  secretToken,
		GLID:         glID,
		GLRepository: glRepository,
		RepoPath:     repoPath,
	}

	serverURL, cleanup := testhelper.NewGitlabTestServer(t, opts)
	defer cleanup()

	gitlabShellDir, cleanup := testhelper.TempDir(t)
	defer cleanup()
	config.Config.GitlabShell.Dir = gitlabShellDir
	config.Config.Gitlab.URL = serverURL
	config.Config.Gitlab.HTTPSettings.User = gitlabUser
	config.Config.Gitlab.HTTPSettings.Password = gitlabPassword
	config.Config.Gitlab.SecretFile = filepath.Join(gitlabShellDir, ".gitlab_shell_secret")

	testhelper.WriteShellSecretFile(t, gitlabShellDir, secretToken)

	locator := config.NewLocator(config.Config)
	txManager := transaction.NewManager(config.Config)
	hookManager := gitalyhook.NewManager(locator, txManager, gitalyhook.GitlabAPIStub, config.Config)
	gitCmdFactory := git.NewExecCommandFactory(config.Config)

	gitalyServer := testhelper.NewServerWithAuth(t, nil, nil, config.Config.Auth.Token)

	gitalypb.RegisterSmartHTTPServiceServer(gitalyServer.GrpcServer(), NewServer(config.Config, locator, gitCmdFactory))
	gitalypb.RegisterHookServiceServer(gitalyServer.GrpcServer(), hook.NewServer(config.Config, hookManager, gitCmdFactory))
	reflection.Register(gitalyServer.GrpcServer())
	gitalyServer.Start(t)
	defer gitalyServer.Stop()

	internalSocket := config.Config.GitalyInternalSocketPath()
	internalListener, err := net.Listen("unix", internalSocket)
	require.NoError(t, err)

	go func() {
		gitalyServer.GrpcServer().Serve(internalListener)
	}()

	client, conn := newSmartHTTPClient(t, "unix://"+gitalyServer.Socket(), config.Config.Auth.Token)
	defer conn.Close()

	stream, err := client.PostReceivePack(ctx)
	require.NoError(t, err)

	push := newTestPush(t, nil)
	request := &gitalypb.PostReceivePackRequest{Repository: repo, GlId: glID, GlRepository: glRepository}
	response := doPush(t, stream, request, push.body)

	expectedResponse := "0049\x01000eunpack ok\n0019ok refs/heads/master\n0019ok refs/heads/branch\n00000000"
	require.Equal(t, expectedResponse, string(response), "Expected response to be %q, got %q", expectedResponse, response)
}

type testTransactionServer struct {
	gitalypb.UnimplementedRefTransactionServer
	called int
}

func (t *testTransactionServer) VoteTransaction(ctx context.Context, in *gitalypb.VoteTransactionRequest) (*gitalypb.VoteTransactionResponse, error) {
	t.called++
	return &gitalypb.VoteTransactionResponse{
		State: gitalypb.VoteTransactionResponse_COMMIT,
	}, nil
}

func TestPostReceiveWithReferenceTransactionHook(t *testing.T) {
	refTransactionServer := &testTransactionServer{}

	locator := config.NewLocator(config.Config)
	txManager := transaction.NewManager(config.Config)
	hookManager := gitalyhook.NewManager(locator, txManager, gitalyhook.GitlabAPIStub, config.Config)
	gitCmdFactory := git.NewExecCommandFactory(config.Config)

	gitalyServer := testhelper.NewTestGrpcServer(t, nil, nil)
	gitalypb.RegisterSmartHTTPServiceServer(gitalyServer, NewServer(config.Config, locator, gitCmdFactory))
	gitalypb.RegisterHookServiceServer(gitalyServer, hook.NewServer(config.Config, hookManager, gitCmdFactory))
	gitalypb.RegisterRefTransactionServer(gitalyServer, refTransactionServer)
	healthpb.RegisterHealthServer(gitalyServer, health.NewServer())
	reflection.Register(gitalyServer)

	gitalySocketPath := testhelper.GetTemporaryGitalySocketFileName(t)
	listener, err := net.Listen("unix", gitalySocketPath)
	require.NoError(t, err)

	internalListener, err := net.Listen("unix", config.Config.GitalyInternalSocketPath())
	require.NoError(t, err)

	go gitalyServer.Serve(listener)
	go gitalyServer.Serve(internalListener)
	defer gitalyServer.Stop()

	client, conn := newSmartHTTPClient(t, "unix://"+gitalySocketPath, config.Config.Auth.Token)
	defer conn.Close()

	// As we ain't got a Praefect server setup, we instead hooked up the
	// RefTransaction server for Gitaly itself. As this is the only Praefect
	// service required in this context, we can just pretend that
	// Gitaly is the Praefect server and inject it.
	praefectServer, err := metadata.PraefectFromConfig(pconfig.Config{
		SocketPath: "unix://" + gitalySocketPath,
	})
	require.NoError(t, err)

	ctx, cancel := testhelper.Context()
	defer cancel()
	ctx, err = metadata.InjectTransaction(ctx, 1234, "primary", true)
	require.NoError(t, err)
	ctx, err = praefectServer.Inject(ctx)
	require.NoError(t, err)
	ctx = helper.IncomingToOutgoing(ctx)

	t.Run("update", func(t *testing.T) {
		stream, err := client.PostReceivePack(ctx)
		require.NoError(t, err)

		repo, _, cleanup := testhelper.NewTestRepo(t)
		defer cleanup()

		request := &gitalypb.PostReceivePackRequest{Repository: repo, GlId: "key-1234", GlRepository: "some_repo"}
		response := doPush(t, stream, request, newTestPush(t, nil).body)

		expectedResponse := "0049\x01000eunpack ok\n0019ok refs/heads/master\n0019ok refs/heads/branch\n00000000"
		require.Equal(t, expectedResponse, string(response), "Expected response to be %q, got %q", expectedResponse, response)
		require.Equal(t, 2, refTransactionServer.called)
	})

	t.Run("delete", func(t *testing.T) {
		refTransactionServer.called = 0

		stream, err := client.PostReceivePack(ctx)
		require.NoError(t, err)

		repo, repoPath, cleanup := testhelper.NewTestRepo(t)
		defer cleanup()

		// Create a new branch which we're about to delete. We also pack references because
		// this used to generate two transactions: one for the packed-refs file and one for
		// the loose ref. We only expect a single transaction though, given that the
		// packed-refs transaction should get filtered out.
		testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "branch", "delete-me")
		testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "pack-refs", "--all")
		branchOID := text.ChompBytes(testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "rev-parse", "refs/heads/delete-me"))

		uploadPackData := &bytes.Buffer{}
		pktline.WriteString(uploadPackData, fmt.Sprintf("%s %s refs/heads/delete-me\x00 %s", branchOID, git.ZeroOID.String(), uploadPackCapabilities))
		pktline.WriteFlush(uploadPackData)

		request := &gitalypb.PostReceivePackRequest{Repository: repo, GlId: "key-1234", GlRepository: "some_repo"}
		response := doPush(t, stream, request, uploadPackData)

		expectedResponse := "0033\x01000eunpack ok\n001cok refs/heads/delete-me\n00000000"
		require.Equal(t, expectedResponse, string(response), "Expected response to be %q, got %q", expectedResponse, response)
		require.Equal(t, 1, refTransactionServer.called)
	})
}
