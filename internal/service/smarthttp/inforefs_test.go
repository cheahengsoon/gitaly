package smarthttp

import (
	"bytes"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	pbhelper "gitlab.com/gitlab-org/gitaly-proto/go/helper"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	scratchDir   = "testdata/scratch"
	testRepoRoot = "testdata/data"
	testRepo     = "group/test.git"
)

var serverSocketPath = path.Join(scratchDir, "gitaly.sock")

func TestMain(m *testing.M) {
	source := "https://gitlab.com/gitlab-org/gitlab-test.git"
	clonePath := path.Join(testRepoRoot, testRepo)
	if _, err := os.Stat(clonePath); err != nil {
		testCmd := exec.Command("git", "clone", "--bare", source, clonePath)
		testCmd.Stdout = os.Stdout
		testCmd.Stderr = os.Stderr

		if err := testCmd.Run(); err != nil {
			log.Printf("Test setup: failed to run %v", testCmd)
			os.Exit(-1)
		}
	}

	if err := os.MkdirAll(scratchDir, 0755); err != nil {
		log.Fatal(err)
	}

	os.Exit(func() int {
		return m.Run()
	}())
}

func TestSuccessfulInfoRefsUploadPack(t *testing.T) {
	server := runSmartHTTPServer(t)
	defer server.Stop()

	client := newSmartHTTPClient(t)
	repo := &pb.Repository{Path: path.Join(testRepoRoot, testRepo)}
	rpcRequest := &pb.InfoRefsRequest{Repository: repo}

	c, err := client.InfoRefsUploadPack(context.Background(), rpcRequest)
	if err != nil {
		t.Fatal(err)
	}

	response := readFullInfoRefsResponse(t, pbhelper.InfoRefsClientWriterTo{c})

	assertGitRefAdvertisement(t, "InfoRefsUploadPack", response.String(), "001e# service=git-upload-pack", "0000", []string{
		"003ef4e6814c3e4e7a0de82a9e7cd20c626cc963a2f8 refs/tags/v1.0.0",
		"00416f6d7e7ed97bb5f0054f2b1df789b39ca89b6ff9 refs/tags/v1.0.0^{}",
	})
}

func TestSuccessfulInfoRefsReceivePack(t *testing.T) {
	server := runSmartHTTPServer(t)
	defer server.Stop()

	client := newSmartHTTPClient(t)
	repo := &pb.Repository{Path: path.Join(testRepoRoot, testRepo)}
	rpcRequest := &pb.InfoRefsRequest{Repository: repo}

	c, err := client.InfoRefsReceivePack(context.Background(), rpcRequest)
	if err != nil {
		t.Fatal(err)
	}

	response := readFullInfoRefsResponse(t, pbhelper.InfoRefsClientWriterTo{c})

	assertGitRefAdvertisement(t, "InfoRefsReceivePack", response.String(), "001f# service=git-receive-pack", "0000", []string{
		"003ef4e6814c3e4e7a0de82a9e7cd20c626cc963a2f8 refs/tags/v1.0.0",
		"003e8a2a6eb295bb170b34c24c76c49ed0e9b2eaf34b refs/tags/v1.1.0",
	})
}

func TestFailureRepoNotFoundInfoRefsReceivePack(t *testing.T) {
	server := runSmartHTTPServer(t)
	defer server.Stop()

	client := newSmartHTTPClient(t)
	repo := &pb.Repository{Path: path.Join(testRepoRoot, "another_repo")}
	rpcRequest := &pb.InfoRefsRequest{Repository: repo}

	c, err := client.InfoRefsReceivePack(context.Background(), rpcRequest)
	if err != nil {
		t.Fatal(err)
	}

	if err := drainInfoRefs(c); !strings.Contains(err.Error(), "testdata/data/another_repo]: exit status 128") {
		t.Fatal(err)
	}
}

func TestFailureRepoNotSetInfoRefsReceivePack(t *testing.T) {
	server := runSmartHTTPServer(t)
	defer server.Stop()

	client := newSmartHTTPClient(t)
	rpcRequest := &pb.InfoRefsRequest{}

	c, err := client.InfoRefsReceivePack(context.Background(), rpcRequest)
	if err != nil {
		t.Fatal(err)
	}

	if err := drainInfoRefs(c); err.Error() != "rpc error: code = 3 desc = GetInfoRefs: repo argument is missing" {
		t.Fatal(err)
	}
}

func runSmartHTTPServer(t *testing.T) *grpc.Server {
	server := grpc.NewServer()
	listener, err := net.Listen("unix", serverSocketPath)
	if err != nil {
		t.Fatal(err)
	}

	pb.RegisterSmartHTTPServer(server, NewServer())
	reflection.Register(server)

	go server.Serve(listener)

	return server
}

func newSmartHTTPClient(t *testing.T) pb.SmartHTTPClient {
	connOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, _ time.Duration) (net.Conn, error) {
			return net.Dial("unix", addr)
		}),
	}
	conn, err := grpc.Dial(serverSocketPath, connOpts...)
	if err != nil {
		t.Fatal(err)
	}

	return pb.NewSmartHTTPClient(conn)
}

func assertGitRefAdvertisement(t *testing.T, rpc, responseBody string, firstLine, lastLine string, middleLines []string) {
	responseLines := strings.Split(responseBody, "\n")

	if responseLines[0] != firstLine {
		t.Errorf("%q: expected response first line to be %q, found %q", rpc, firstLine, responseLines[0])
	}

	lastIndex := len(responseLines) - 1
	if responseLines[lastIndex] != lastLine {
		t.Errorf("%q: expected response last line to be %q, found %q", rpc, lastLine, responseLines[lastIndex])
	}

	for _, ref := range middleLines {
		if !strings.Contains(responseBody, ref) {
			t.Errorf("%q: expected response to contain %q, found none", rpc, ref)
		}
	}
}

func readFullInfoRefsResponse(t *testing.T, c pbhelper.InfoRefsClientWriterTo) *bytes.Buffer {
	buffer := &bytes.Buffer{}
	if _, err := c.WriteTo(buffer); err != nil {
		t.Fatal(err)
	}
	return buffer
}

func drainInfoRefs(c pbhelper.InfoRefsClient) error {
	_, err := (&pbhelper.InfoRefsClientWriterTo{c}).WriteTo(ioutil.Discard)
	return err
}
