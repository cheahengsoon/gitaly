package ref

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/v14/internal/git"
	"gitlab.com/gitlab-org/gitaly/v14/internal/git/catfile"
	"gitlab.com/gitlab-org/gitaly/v14/internal/git/gittest"
	"gitlab.com/gitlab-org/gitaly/v14/internal/git/localrepo"
	"gitlab.com/gitlab-org/gitaly/v14/internal/git/updateref"
	"gitlab.com/gitlab-org/gitaly/v14/internal/helper"
	"gitlab.com/gitlab-org/gitaly/v14/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/v14/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/v14/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func containsRef(refs [][]byte, ref string) bool {
	for _, b := range refs {
		if string(b) == ref {
			return true
		}
	}
	return false
}

func TestSuccessfulFindAllBranchNames(t *testing.T) {
	_, repo, _, client := setupRefService(t)

	rpcRequest := &gitalypb.FindAllBranchNamesRequest{Repository: repo}

	ctx, cancel := testhelper.Context()
	defer cancel()
	c, err := client.FindAllBranchNames(ctx, rpcRequest)
	require.NoError(t, err)

	var names [][]byte
	for {
		r, err := c.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		names = append(names, r.GetNames()...)
	}

	expectedBranches := testhelper.MustReadFile(t, "testdata/branches.txt")
	for _, branch := range bytes.Split(bytes.TrimSpace(expectedBranches), []byte("\n")) {
		require.Contains(t, names, branch)
	}
}

func TestFindAllBranchNamesVeryLargeResponse(t *testing.T) {
	cfg, repoProto, _, client := setupRefService(t)

	ctx, cancel := testhelper.Context()
	defer cancel()

	repo := localrepo.NewTestRepo(t, cfg, repoProto)
	updater, err := updateref.New(ctx, cfg, repo)
	require.NoError(t, err)

	// We want to create enough refs to overflow the default bufio.Scanner
	// buffer. Such an overflow will cause scanner.Bytes() to become invalid
	// at some point. That is expected behavior, but our tests did not
	// trigger it, so we got away with naively using scanner.Bytes() and
	// causing a bug: https://gitlab.com/gitlab-org/gitaly/issues/1473.
	refSizeLowerBound := 100
	numRefs := 2 * bufio.MaxScanTokenSize / refSizeLowerBound

	var testRefs []string
	for i := 0; i < numRefs; i++ {
		refName := fmt.Sprintf("refs/heads/test-%0100d", i)
		require.True(t, len(refName) > refSizeLowerBound, "ref %q must be larger than %d", refName, refSizeLowerBound)

		require.NoError(t, updater.Create(git.ReferenceName(refName), "HEAD"))
		testRefs = append(testRefs, refName)
	}

	require.NoError(t, updater.Commit())

	rpcRequest := &gitalypb.FindAllBranchNamesRequest{Repository: repoProto}

	c, err := client.FindAllBranchNames(ctx, rpcRequest)
	require.NoError(t, err)

	var names [][]byte
	for {
		r, err := c.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		names = append(names, r.GetNames()...)
	}

	for _, branch := range testRefs {
		require.Contains(t, names, []byte(branch), "branch missing from response: %q", branch)
	}
}

func TestEmptyFindAllBranchNamesRequest(t *testing.T) {
	_, client := setupRefServiceWithoutRepo(t)

	rpcRequest := &gitalypb.FindAllBranchNamesRequest{}

	ctx, cancel := testhelper.Context()
	defer cancel()
	c, err := client.FindAllBranchNames(ctx, rpcRequest)
	require.NoError(t, err)

	var recvError error
	for recvError == nil {
		_, recvError = c.Recv()
	}

	if helper.GrpcCode(recvError) != codes.InvalidArgument {
		t.Fatal(recvError)
	}
}

func TestInvalidRepoFindAllBranchNamesRequest(t *testing.T) {
	_, client := setupRefServiceWithoutRepo(t)

	repo := &gitalypb.Repository{StorageName: "default", RelativePath: "made/up/path"}
	rpcRequest := &gitalypb.FindAllBranchNamesRequest{Repository: repo}

	ctx, cancel := testhelper.Context()
	defer cancel()
	c, err := client.FindAllBranchNames(ctx, rpcRequest)
	require.NoError(t, err)

	var recvError error
	for recvError == nil {
		_, recvError = c.Recv()
	}

	if helper.GrpcCode(recvError) != codes.NotFound {
		t.Fatal(recvError)
	}
}

func TestSuccessfulFindAllTagNames(t *testing.T) {
	_, repo, _, client := setupRefService(t)

	rpcRequest := &gitalypb.FindAllTagNamesRequest{Repository: repo}

	ctx, cancel := testhelper.Context()
	defer cancel()
	c, err := client.FindAllTagNames(ctx, rpcRequest)
	require.NoError(t, err)

	var names [][]byte
	for {
		r, err := c.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, r.GetNames()...)
	}

	for _, tag := range []string{"v1.0.0", "v1.1.0"} {
		if !containsRef(names, "refs/tags/"+tag) {
			t.Fatal("Expected to find tag", tag, "in all tag names")
		}
	}
}

func TestEmptyFindAllTagNamesRequest(t *testing.T) {
	_, client := setupRefServiceWithoutRepo(t)

	rpcRequest := &gitalypb.FindAllTagNamesRequest{}

	ctx, cancel := testhelper.Context()
	defer cancel()
	c, err := client.FindAllTagNames(ctx, rpcRequest)
	require.NoError(t, err)

	var recvError error
	for recvError == nil {
		_, recvError = c.Recv()
	}

	if helper.GrpcCode(recvError) != codes.InvalidArgument {
		t.Fatal(recvError)
	}
}

func TestInvalidRepoFindAllTagNamesRequest(t *testing.T) {
	_, client := setupRefServiceWithoutRepo(t)

	repo := &gitalypb.Repository{StorageName: "default", RelativePath: "made/up/path"}
	rpcRequest := &gitalypb.FindAllTagNamesRequest{Repository: repo}

	ctx, cancel := testhelper.Context()
	defer cancel()
	c, err := client.FindAllTagNames(ctx, rpcRequest)
	require.NoError(t, err)

	var recvError error
	for recvError == nil {
		_, recvError = c.Recv()
	}

	if helper.GrpcCode(recvError) != codes.NotFound {
		t.Fatal(recvError)
	}
}

func TestSuccessfulFindDefaultBranchName(t *testing.T) {
	cfg, repo, repoPath, client := setupRefService(t)
	rpcRequest := &gitalypb.FindDefaultBranchNameRequest{Repository: repo}

	// The testing repository has no main branch, so we create it and update
	// HEAD to it
	gittest.Exec(t, cfg, "-C", repoPath, "update-ref", "refs/heads/main", "1a0b36b3cdad1d2ee32457c102a8c0b7056fa863")
	gittest.Exec(t, cfg, "-C", repoPath, "symbolic-ref", "HEAD", "refs/heads/main")

	ctx, cancel := testhelper.Context()
	defer cancel()
	r, err := client.FindDefaultBranchName(ctx, rpcRequest)
	require.NoError(t, err)

	require.Equal(t, git.ReferenceName(r.GetName()), git.DefaultRef)
}

func TestSuccessfulFindDefaultBranchNameLegacy(t *testing.T) {
	_, repo, _, client := setupRefService(t)
	rpcRequest := &gitalypb.FindDefaultBranchNameRequest{Repository: repo}

	ctx, cancel := testhelper.Context()
	defer cancel()
	r, err := client.FindDefaultBranchName(ctx, rpcRequest)
	require.NoError(t, err)

	require.Equal(t, git.ReferenceName(r.GetName()), git.LegacyDefaultRef)
}

func TestEmptyFindDefaultBranchNameRequest(t *testing.T) {
	_, client := setupRefServiceWithoutRepo(t)
	rpcRequest := &gitalypb.FindDefaultBranchNameRequest{}

	ctx, cancel := testhelper.Context()
	defer cancel()
	_, err := client.FindDefaultBranchName(ctx, rpcRequest)

	if helper.GrpcCode(err) != codes.InvalidArgument {
		t.Fatal(err)
	}
}

func TestInvalidRepoFindDefaultBranchNameRequest(t *testing.T) {
	cfg, client := setupRefServiceWithoutRepo(t)
	repo := &gitalypb.Repository{StorageName: cfg.Storages[0].Name, RelativePath: "/made/up/path"}
	rpcRequest := &gitalypb.FindDefaultBranchNameRequest{Repository: repo}

	ctx, cancel := testhelper.Context()
	defer cancel()
	_, err := client.FindDefaultBranchName(ctx, rpcRequest)

	if helper.GrpcCode(err) != codes.NotFound {
		t.Fatal(err)
	}
}

func TestSuccessfulFindLocalBranches(t *testing.T) {
	_, repo, _, client := setupRefService(t)

	rpcRequest := &gitalypb.FindLocalBranchesRequest{Repository: repo}

	ctx, cancel := testhelper.Context()
	defer cancel()
	c, err := client.FindLocalBranches(ctx, rpcRequest)
	require.NoError(t, err)

	var branches []*gitalypb.FindLocalBranchResponse
	for {
		r, err := c.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		branches = append(branches, r.GetBranches()...)
	}

	for name, target := range localBranches {
		localBranch := &gitalypb.FindLocalBranchResponse{
			Name:          []byte(name),
			CommitId:      target.Id,
			CommitSubject: target.Subject,
			CommitAuthor: &gitalypb.FindLocalBranchCommitAuthor{
				Name:  target.Author.Name,
				Email: target.Author.Email,
				Date:  target.Author.Date,
			},
			CommitCommitter: &gitalypb.FindLocalBranchCommitAuthor{
				Name:  target.Committer.Name,
				Email: target.Committer.Email,
				Date:  target.Committer.Date,
			},
			Commit: target,
		}

		assertContainsLocalBranch(t, branches, localBranch)
	}
}

func TestFindLocalBranches_huge_committer(t *testing.T) {
	testhelper.NewFeatureSets(featureflag.ExactPaginationTokenMatch).Run(t, testFindLocalBranchesHugeCommitter)
}

func testFindLocalBranchesHugeCommitter(t *testing.T, ctx context.Context) {
	cfg, repo, repoPath, client := setupRefService(t)

	gittest.WriteCommit(t, cfg, repoPath,
		gittest.WithBranch("refs/heads/improve/awesome"),
		gittest.WithCommitterName(strings.Repeat("A", 100000)),
	)

	rpcRequest := &gitalypb.FindLocalBranchesRequest{Repository: repo}

	c, err := client.FindLocalBranches(ctx, rpcRequest)
	require.NoError(t, err)

	for {
		_, err := c.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}
}

func TestFindLocalBranchesPagination(t *testing.T) {
	testhelper.NewFeatureSets(featureflag.ExactPaginationTokenMatch).Run(t, testFindLocalBranchesPagination)
}

func testFindLocalBranchesPagination(t *testing.T, ctx context.Context) {
	_, repo, _, client := setupRefService(t)

	limit := 1
	rpcRequest := &gitalypb.FindLocalBranchesRequest{
		Repository: repo,
		PaginationParams: &gitalypb.PaginationParameter{
			Limit:     int32(limit),
			PageToken: "refs/heads/gitaly/squash-test",
		},
	}
	c, err := client.FindLocalBranches(ctx, rpcRequest)
	require.NoError(t, err)

	var branches []*gitalypb.FindLocalBranchResponse
	for {
		r, err := c.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		branches = append(branches, r.GetBranches()...)
	}

	require.Len(t, branches, limit)

	expectedBranch := "refs/heads/improve/awesome"
	target := localBranches[expectedBranch]

	branch := &gitalypb.FindLocalBranchResponse{
		Name:          []byte(expectedBranch),
		CommitId:      target.Id,
		CommitSubject: target.Subject,
		CommitAuthor: &gitalypb.FindLocalBranchCommitAuthor{
			Name:  target.Author.Name,
			Email: target.Author.Email,
			Date:  target.Author.Date,
		},
		CommitCommitter: &gitalypb.FindLocalBranchCommitAuthor{
			Name:  target.Committer.Name,
			Email: target.Committer.Email,
			Date:  target.Committer.Date,
		},
		Commit: target,
	}
	assertContainsLocalBranch(t, branches, branch)
}

func TestFindLocalBranchesPaginationSequence(t *testing.T) {
	testhelper.NewFeatureSets(featureflag.ExactPaginationTokenMatch).Run(t, testFindLocalBranchesPaginationSequence)
}

func testFindLocalBranchesPaginationSequence(t *testing.T, ctx context.Context) {
	_, repo, _, client := setupRefService(t)

	limit := 2
	firstRPCRequest := &gitalypb.FindLocalBranchesRequest{
		Repository: repo,
		PaginationParams: &gitalypb.PaginationParameter{
			Limit: int32(limit),
		},
	}
	c, err := client.FindLocalBranches(ctx, firstRPCRequest)
	require.NoError(t, err)

	var firstResponseBranches []*gitalypb.FindLocalBranchResponse
	for {
		r, err := c.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		firstResponseBranches = append(firstResponseBranches, r.GetBranches()...)
	}

	require.Len(t, firstResponseBranches, limit)

	secondRPCRequest := &gitalypb.FindLocalBranchesRequest{
		Repository: repo,
		PaginationParams: &gitalypb.PaginationParameter{
			Limit:     1,
			PageToken: string(firstResponseBranches[0].Name),
		},
	}
	c, err = client.FindLocalBranches(ctx, secondRPCRequest)
	require.NoError(t, err)

	var secondResponseBranches []*gitalypb.FindLocalBranchResponse
	for {
		r, err := c.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		secondResponseBranches = append(secondResponseBranches, r.GetBranches()...)
	}

	require.Len(t, secondResponseBranches, 1)
	require.Equal(t, firstResponseBranches[1], secondResponseBranches[0])
}

func TestFindLocalBranchesPaginationWithIncorrectToken(t *testing.T) {
	testhelper.NewFeatureSets(featureflag.ExactPaginationTokenMatch).Run(t, testFindLocalBranchesPaginationWithIncorrectToken)
}

func testFindLocalBranchesPaginationWithIncorrectToken(t *testing.T, ctx context.Context) {
	_, repo, _, client := setupRefService(t)

	limit := 1
	rpcRequest := &gitalypb.FindLocalBranchesRequest{
		Repository: repo,
		PaginationParams: &gitalypb.PaginationParameter{
			Limit:     int32(limit),
			PageToken: "refs/heads/random-unknown-branch",
		},
	}
	c, err := client.FindLocalBranches(ctx, rpcRequest)
	require.NoError(t, err)

	if featureflag.ExactPaginationTokenMatch.IsEnabled(ctx) {
		_, err = c.Recv()
		require.NotEqual(t, err, io.EOF)
		testhelper.RequireGrpcError(t, helper.ErrInternalf("could not find page token"), err)
	} else {
		require.NoError(t, err)

		var branches []*gitalypb.FindLocalBranchResponse
		for {
			r, err := c.Recv()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			branches = append(branches, r.GetBranches()...)
		}

		testhelper.ProtoEqual(t, []*gitalypb.FindLocalBranchResponse{
			{
				Name:          []byte("refs/heads/rebase-encoding-failure-trigger"),
				Commit:        gittest.CommitsByID["ca47bfd5e930148c42ed74c3b561a8783e381f7f"],
				CommitId:      "ca47bfd5e930148c42ed74c3b561a8783e381f7f",
				CommitSubject: []byte("Add Modula-2 source file for language detection"),
				CommitAuthor: &gitalypb.FindLocalBranchCommitAuthor{
					Name:     []byte("Jacob Vosmaer"),
					Email:    []byte("jacob@gitlab.com"),
					Date:     &timestamppb.Timestamp{Seconds: 1501503403},
					Timezone: []byte("+0200"),
				},
				CommitCommitter: &gitalypb.FindLocalBranchCommitAuthor{
					Name:     []byte("Ahmad Sherif"),
					Email:    []byte("me@ahmadsherif.com"),
					Date:     &timestamppb.Timestamp{Seconds: 1521033060},
					Timezone: []byte("+0100"),
				},
			},
		}, branches)
	}
}

// Test that `s` contains the elements in `relativeOrder` in that order
// (relative to each other)
func isOrderedSubset(subset, set []string) bool {
	subsetIndex := 0 // The string we are currently looking for from `subset`
	for _, element := range set {
		if element != subset[subsetIndex] {
			continue
		}

		subsetIndex++

		if subsetIndex == len(subset) { // We found all elements in that order
			return true
		}
	}
	return false
}

func TestFindLocalBranchesSort(t *testing.T) {
	testhelper.NewFeatureSets(featureflag.ExactPaginationTokenMatch).Run(t, testFindLocalBranchesSort)
}

func testFindLocalBranchesSort(t *testing.T, ctx context.Context) {
	testCases := []struct {
		desc          string
		relativeOrder []string
		sortBy        gitalypb.FindLocalBranchesRequest_SortBy
	}{
		{
			desc:          "In ascending order by name",
			relativeOrder: []string{"refs/heads/'test'", "refs/heads/100%branch", "refs/heads/improve/awesome", "refs/heads/master"},
			sortBy:        gitalypb.FindLocalBranchesRequest_NAME,
		},
		{
			desc:          "In ascending order by commiter date",
			relativeOrder: []string{"refs/heads/improve/awesome", "refs/heads/'test'", "refs/heads/100%branch", "refs/heads/master"},
			sortBy:        gitalypb.FindLocalBranchesRequest_UPDATED_ASC,
		},
		{
			desc:          "In descending order by commiter date",
			relativeOrder: []string{"refs/heads/master", "refs/heads/100%branch", "refs/heads/'test'", "refs/heads/improve/awesome"},
			sortBy:        gitalypb.FindLocalBranchesRequest_UPDATED_DESC,
		},
	}

	_, repo, _, client := setupRefService(t)

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			rpcRequest := &gitalypb.FindLocalBranchesRequest{Repository: repo, SortBy: testCase.sortBy}

			c, err := client.FindLocalBranches(ctx, rpcRequest)
			require.NoError(t, err)

			var branches []string
			for {
				r, err := c.Recv()
				if err == io.EOF {
					break
				}
				require.NoError(t, err)

				for _, branch := range r.GetBranches() {
					branches = append(branches, string(branch.Name))
				}
			}

			if !isOrderedSubset(testCase.relativeOrder, branches) {
				t.Fatalf("%s: Expected branches to have relative order %v; got them as %v", testCase.desc, testCase.relativeOrder, branches)
			}
		})
	}
}

func TestEmptyFindLocalBranchesRequest(t *testing.T) {
	_, client := setupRefServiceWithoutRepo(t)

	rpcRequest := &gitalypb.FindLocalBranchesRequest{}

	ctx, cancel := testhelper.Context()
	defer cancel()
	c, err := client.FindLocalBranches(ctx, rpcRequest)
	require.NoError(t, err)

	var recvError error
	for recvError == nil {
		_, recvError = c.Recv()
	}

	if helper.GrpcCode(recvError) != codes.InvalidArgument {
		t.Fatal(recvError)
	}
}

func TestSuccessfulFindAllBranchesRequest(t *testing.T) {
	cfg, repo, repoPath, client := setupRefService(t)

	remoteBranch := &gitalypb.FindAllBranchesResponse_Branch{
		Name: []byte("refs/remotes/origin/fake-remote-branch"),
		Target: &gitalypb.GitCommit{
			Id:        "913c66a37b4a45b9769037c55c2d238bd0942d2e",
			Subject:   []byte("Files, encoding and much more"),
			Body:      []byte("Files, encoding and much more\n\nSigned-off-by: Dmitriy Zaporozhets <dmitriy.zaporozhets@gmail.com>\n"),
			BodySize:  98,
			ParentIds: []string{"cfe32cf61b73a0d5e9f13e774abde7ff789b1660"},
			Author: &gitalypb.CommitAuthor{
				Name:     []byte("Dmitriy Zaporozhets"),
				Email:    []byte("dmitriy.zaporozhets@gmail.com"),
				Date:     &timestamppb.Timestamp{Seconds: 1393488896},
				Timezone: []byte("+0200"),
			},
			Committer: &gitalypb.CommitAuthor{
				Name:     []byte("Dmitriy Zaporozhets"),
				Email:    []byte("dmitriy.zaporozhets@gmail.com"),
				Date:     &timestamppb.Timestamp{Seconds: 1393488896},
				Timezone: []byte("+0200"),
			},
			SignatureType: gitalypb.SignatureType_PGP,
			TreeId:        "faafbe7fe23fb83c664c78aaded9566c8f934412",
		},
	}

	gittest.WriteRef(t, cfg, repoPath, "refs/remotes/origin/fake-remote-branch", git.ObjectID(remoteBranch.Target.Id))

	request := &gitalypb.FindAllBranchesRequest{Repository: repo}
	ctx, cancel := testhelper.Context()
	defer cancel()
	c, err := client.FindAllBranches(ctx, request)
	require.NoError(t, err)

	branches := readFindAllBranchesResponsesFromClient(t, c)

	// It contains local branches
	for name, target := range localBranches {
		branch := &gitalypb.FindAllBranchesResponse_Branch{
			Name:   []byte(name),
			Target: target,
		}
		assertContainsBranch(t, branches, branch)
	}

	// It contains our fake remote branch
	assertContainsBranch(t, branches, remoteBranch)
}

func TestSuccessfulFindAllBranchesRequestWithMergedBranches(t *testing.T) {
	cfg, repoProto, repoPath, client := setupRefService(t)

	repo := localrepo.NewTestRepo(t, cfg, repoProto)

	ctx, cancel := testhelper.Context()
	defer cancel()

	localRefs := gittest.Exec(t, cfg, "-C", repoPath, "for-each-ref", "--format=%(refname:strip=2)", "refs/heads")
	for _, ref := range strings.Split(string(localRefs), "\n") {
		ref = strings.TrimSpace(ref)
		if _, ok := localBranches["refs/heads/"+ref]; ok || ref == "master" || ref == "" {
			continue
		}
		gittest.Exec(t, cfg, "-C", repoPath, "branch", "-D", ref)
	}

	expectedRefs := []string{"refs/heads/100%branch", "refs/heads/improve/awesome", "refs/heads/'test'"}

	var expectedBranches []*gitalypb.FindAllBranchesResponse_Branch
	for _, name := range expectedRefs {
		target, ok := localBranches[name]
		require.True(t, ok)

		branch := &gitalypb.FindAllBranchesResponse_Branch{
			Name:   []byte(name),
			Target: target,
		}
		expectedBranches = append(expectedBranches, branch)
	}

	masterCommit, err := repo.ReadCommit(ctx, "master")
	require.NoError(t, err)
	expectedBranches = append(expectedBranches, &gitalypb.FindAllBranchesResponse_Branch{
		Name:   []byte("refs/heads/master"),
		Target: masterCommit,
	})

	testCases := []struct {
		desc             string
		request          *gitalypb.FindAllBranchesRequest
		expectedBranches []*gitalypb.FindAllBranchesResponse_Branch
	}{
		{
			desc: "all merged branches",
			request: &gitalypb.FindAllBranchesRequest{
				Repository: repoProto,
				MergedOnly: true,
			},
			expectedBranches: expectedBranches,
		},
		{
			desc: "all merged from a list of branches",
			request: &gitalypb.FindAllBranchesRequest{
				Repository: repoProto,
				MergedOnly: true,
				MergedBranches: [][]byte{
					[]byte("refs/heads/100%branch"),
					[]byte("refs/heads/improve/awesome"),
					[]byte("refs/heads/gitaly-stuff"),
				},
			},
			expectedBranches: expectedBranches[:2],
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			c, err := client.FindAllBranches(ctx, testCase.request)
			require.NoError(t, err)

			branches := readFindAllBranchesResponsesFromClient(t, c)
			require.Len(t, branches, len(testCase.expectedBranches))

			for _, branch := range branches {
				// The GitCommit object returned by GetCommit() above and the one returned in the response
				// vary a lot. We can't guarantee that master will be fixed at a certain commit so we can't create
				// a structure for it manually, hence this hack.
				if string(branch.Name) == "refs/heads/master" {
					continue
				}

				assertContainsBranch(t, testCase.expectedBranches, branch)
			}
		})
	}
}

func TestInvalidFindAllBranchesRequest(t *testing.T) {
	_, client := setupRefServiceWithoutRepo(t)

	testCases := []struct {
		description string
		request     *gitalypb.FindAllBranchesRequest
	}{
		{
			description: "Empty request",
			request:     &gitalypb.FindAllBranchesRequest{},
		},
		{
			description: "Invalid repo",
			request: &gitalypb.FindAllBranchesRequest{
				Repository: &gitalypb.Repository{
					StorageName:  "fake",
					RelativePath: "repo",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()
			c, err := client.FindAllBranches(ctx, tc.request)
			require.NoError(t, err)

			var recvError error
			for recvError == nil {
				_, recvError = c.Recv()
			}

			testhelper.RequireGrpcCode(t, recvError, codes.InvalidArgument)
		})
	}
}

func readFindAllBranchesResponsesFromClient(t *testing.T, c gitalypb.RefService_FindAllBranchesClient) (branches []*gitalypb.FindAllBranchesResponse_Branch) {
	for {
		r, err := c.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		branches = append(branches, r.GetBranches()...)
	}

	return
}

func TestListTagNamesContainingCommit(t *testing.T) {
	_, repoProto, _, client := setupRefService(t)

	testCases := []struct {
		description string
		commitID    string
		code        codes.Code
		limit       uint32
		tags        []string
	}{
		{
			description: "no commit ID",
			commitID:    "",
			code:        codes.InvalidArgument,
		},
		{
			description: "current master HEAD",
			commitID:    "e63f41fe459e62e1228fcef60d7189127aeba95a",
			code:        codes.OK,
			tags:        []string{},
		},
		{
			description: "init commit",
			commitID:    "1a0b36b3cdad1d2ee32457c102a8c0b7056fa863",
			code:        codes.OK,
			tags:        []string{"v1.0.0", "v1.1.0"},
		},
		{
			description: "limited response size",
			commitID:    "1a0b36b3cdad1d2ee32457c102a8c0b7056fa863",
			code:        codes.OK,
			limit:       1,
			tags:        []string{"v1.0.0"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			request := &gitalypb.ListTagNamesContainingCommitRequest{Repository: repoProto, CommitId: tc.commitID}

			c, err := client.ListTagNamesContainingCommit(ctx, request)
			require.NoError(t, err)

			var names []string
			for {
				r, err := c.Recv()
				if err == io.EOF {
					break
				} else if tc.code != codes.OK {
					testhelper.RequireGrpcCode(t, err, tc.code)

					return
				}
				require.NoError(t, err)

				for _, name := range r.GetTagNames() {
					names = append(names, string(name))
				}
			}

			// Test for inclusion instead of equality because new refs
			// will get added to the gitlab-test repo over time.
			require.Subset(t, names, tc.tags)
		})
	}
}

func TestListBranchNamesContainingCommit(t *testing.T) {
	_, repo, _, client := setupRefService(t)

	testCases := []struct {
		description string
		commitID    string
		code        codes.Code
		limit       uint32
		branches    []string
	}{
		{
			description: "no commit ID",
			commitID:    "",
			code:        codes.InvalidArgument,
		},
		{
			description: "current master HEAD",
			commitID:    "e63f41fe459e62e1228fcef60d7189127aeba95a",
			code:        codes.OK,
			branches:    []string{"master"},
		},
		{
			// gitlab-test contains a branch refs/heads/1942eed5cc108b19c7405106e81fa96125d0be22
			// which is in conflift with a commit with the same ID
			description: "branch name is also commit id",
			commitID:    "1942eed5cc108b19c7405106e81fa96125d0be22",
			code:        codes.OK,
			branches:    []string{"1942eed5cc108b19c7405106e81fa96125d0be22"},
		},
		{
			description: "init commit",
			commitID:    "1a0b36b3cdad1d2ee32457c102a8c0b7056fa863",
			code:        codes.OK,
			branches: []string{
				"deleted-image-test",
				"ends-with.json",
				"master",
				"conflict-non-utf8",
				"'test'",
				"ʕ•ᴥ•ʔ",
				"'test'",
				"100%branch",
			},
		},
		{
			description: "init commit",
			commitID:    "1a0b36b3cdad1d2ee32457c102a8c0b7056fa863",
			code:        codes.OK,
			limit:       1,
			branches:    []string{"'test'"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			request := &gitalypb.ListBranchNamesContainingCommitRequest{Repository: repo, CommitId: tc.commitID}

			c, err := client.ListBranchNamesContainingCommit(ctx, request)
			require.NoError(t, err)

			var names []string
			for {
				r, err := c.Recv()
				if err == io.EOF {
					break
				} else if tc.code != codes.OK {
					testhelper.RequireGrpcCode(t, err, tc.code)

					return
				}
				require.NoError(t, err)

				for _, name := range r.GetBranchNames() {
					names = append(names, string(name))
				}
			}

			// Test for inclusion instead of equality because new refs
			// will get added to the gitlab-test repo over time.
			require.Subset(t, names, tc.branches)
		})
	}
}

func TestSuccessfulFindTagRequest(t *testing.T) {
	cfg, repoProto, repoPath, client := setupRefService(t)

	repo := localrepo.NewTestRepo(t, cfg, repoProto)

	blobID := git.ObjectID("faaf198af3a36dbf41961466703cc1d47c61d051")
	commitID := git.ObjectID("6f6d7e7ed97bb5f0054f2b1df789b39ca89b6ff9")

	gitCommit := testhelper.GitLabTestCommit(commitID.String())

	ctx, cancel := testhelper.Context()
	defer cancel()

	bigCommitID := gittest.WriteCommit(t, cfg, repoPath,
		gittest.WithBranch("local-big-commits"),
		gittest.WithMessage("An empty commit with REALLY BIG message\n\n"+strings.Repeat("a", helper.MaxCommitOrTagMessageSize+1)),
		gittest.WithParents("60ecb67744cb56576c30214ff52294f8ce2def98"),
	)
	bigCommit, err := repo.ReadCommit(ctx, git.Revision(bigCommitID))
	require.NoError(t, err)

	annotatedTagID := gittest.WriteTag(t, cfg, repoPath, "v1.2.0", blobID.Revision(), gittest.WriteTagConfig{Message: "Blob tag"})

	gittest.WriteTag(t, cfg, repoPath, "v1.3.0", commitID.Revision())
	gittest.WriteTag(t, cfg, repoPath, "v1.4.0", blobID.Revision())

	// To test recursive resolving to a commit
	gittest.WriteTag(t, cfg, repoPath, "v1.5.0", "v1.3.0")

	// A tag to commit with a big message
	gittest.WriteTag(t, cfg, repoPath, "v1.6.0", bigCommitID.Revision())

	// A tag with a big message
	bigMessage := strings.Repeat("a", 11*1024)
	bigMessageTag1ID := gittest.WriteTag(t, cfg, repoPath, "v1.7.0", commitID.Revision(), gittest.WriteTagConfig{Message: bigMessage})

	// A tag with a commit id as its name
	commitTagID := gittest.WriteTag(t, cfg, repoPath, commitID.String(), commitID.Revision(), gittest.WriteTagConfig{Message: "commit tag with a commit sha as the name"})

	// a tag of a tag
	tagOfTagID := gittest.WriteTag(t, cfg, repoPath, "tag-of-tag", commitTagID.Revision(), gittest.WriteTagConfig{Message: "tag of a tag"})

	expectedTags := []*gitalypb.Tag{
		{
			Name:         []byte(commitID),
			Id:           commitTagID.String(),
			TargetCommit: gitCommit,
			Message:      []byte("commit tag with a commit sha as the name"),
			MessageSize:  40,
			Tagger: &gitalypb.CommitAuthor{
				Name:     []byte("Scrooge McDuck"),
				Email:    []byte("scrooge@mcduck.com"),
				Date:     &timestamppb.Timestamp{Seconds: 1572776879},
				Timezone: []byte("+0100"),
			},
		},
		{
			Name:         []byte("tag-of-tag"),
			Id:           tagOfTagID.String(),
			TargetCommit: gitCommit,
			Message:      []byte("tag of a tag"),
			MessageSize:  12,
			Tagger: &gitalypb.CommitAuthor{
				Name:     []byte("Scrooge McDuck"),
				Email:    []byte("scrooge@mcduck.com"),
				Date:     &timestamppb.Timestamp{Seconds: 1572776879},
				Timezone: []byte("+0100"),
			},
		},
		{
			Name:         []byte("v1.0.0"),
			Id:           "f4e6814c3e4e7a0de82a9e7cd20c626cc963a2f8",
			TargetCommit: gitCommit,
			Message:      []byte("Release"),
			MessageSize:  7,
			Tagger: &gitalypb.CommitAuthor{
				Name:     []byte("Dmitriy Zaporozhets"),
				Email:    []byte("dmitriy.zaporozhets@gmail.com"),
				Date:     &timestamppb.Timestamp{Seconds: 1393491299},
				Timezone: []byte("+0200"),
			},
			SignatureType: gitalypb.SignatureType_NONE,
		},
		{
			Name:         []byte("v1.1.0"),
			Id:           "8a2a6eb295bb170b34c24c76c49ed0e9b2eaf34b",
			TargetCommit: testhelper.GitLabTestCommit("5937ac0a7beb003549fc5fd26fc247adbce4a52e"),
			Message:      []byte("Version 1.1.0"),
			MessageSize:  13,
			Tagger: &gitalypb.CommitAuthor{
				Name:     []byte("Dmitriy Zaporozhets"),
				Email:    []byte("dmitriy.zaporozhets@gmail.com"),
				Date:     &timestamppb.Timestamp{Seconds: 1393505709},
				Timezone: []byte("+0200"),
			},
		},
		{
			Name:         []byte("v1.1.1"),
			Id:           "8f03acbcd11c53d9c9468078f32a2622005a4841",
			TargetCommit: testhelper.GitLabTestCommit("189a6c924013fc3fe40d6f1ec1dc20214183bc97"),
			Message:      []byte("x509 signed tag\n-----BEGIN SIGNED MESSAGE-----\nMIISfwYJKoZIhvcNAQcCoIIScDCCEmwCAQExDTALBglghkgBZQMEAgEwCwYJKoZI\nhvcNAQcBoIIP8zCCB3QwggVcoAMCAQICBBXXLOIwDQYJKoZIhvcNAQELBQAwgbYx\nCzAJBgNVBAYTAkRFMQ8wDQYDVQQIDAZCYXllcm4xETAPBgNVBAcMCE11ZW5jaGVu\nMRAwDgYDVQQKDAdTaWVtZW5zMREwDwYDVQQFEwhaWlpaWlpBNjEdMBsGA1UECwwU\nU2llbWVucyBUcnVzdCBDZW50ZXIxPzA9BgNVBAMMNlNpZW1lbnMgSXNzdWluZyBD\nQSBNZWRpdW0gU3RyZW5ndGggQXV0aGVudGljYXRpb24gMjAxNjAeFw0xNzAyMDMw\nNjU4MzNaFw0yMDAyMDMwNjU4MzNaMFsxETAPBgNVBAUTCFowMDBOV0RIMQ4wDAYD\nVQQqDAVSb2dlcjEOMAwGA1UEBAwFTWVpZXIxEDAOBgNVBAoMB1NpZW1lbnMxFDAS\nBgNVBAMMC01laWVyIFJvZ2VyMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKC\nAQEAuBNea/68ZCnHYQjpm/k3ZBG0wBpEKSwG6lk9CEQlSxsqVLQHAoAKBIlJm1in\nYVLcK/Sq1yhYJ/qWcY/M53DhK2rpPuhtrWJUdOUy8EBWO20F4bd4Fw9pO7jt8bme\nu33TSrK772vKjuppzB6SeG13Cs08H+BIeD106G27h7ufsO00pvsxoSDL+uc4slnr\npBL+2TAL7nSFnB9QHWmRIK27SPqJE+lESdb0pse11x1wjvqKy2Q7EjL9fpqJdHzX\nNLKHXd2r024TOORTa05DFTNR+kQEKKV96XfpYdtSBomXNQ44cisiPBJjFtYvfnFE\nwgrHa8fogn/b0C+A+HAoICN12wIDAQABo4IC4jCCAt4wHQYDVR0OBBYEFCF+gkUp\nXQ6xGc0kRWXuDFxzA14zMEMGA1UdEQQ8MDqgIwYKKwYBBAGCNxQCA6AVDBNyLm1l\naWVyQHNpZW1lbnMuY29tgRNyLm1laWVyQHNpZW1lbnMuY29tMA4GA1UdDwEB/wQE\nAwIHgDAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYBBQUHAwQwgcoGA1UdHwSBwjCB\nvzCBvKCBuaCBtoYmaHR0cDovL2NoLnNpZW1lbnMuY29tL3BraT9aWlpaWlpBNi5j\ncmyGQWxkYXA6Ly9jbC5zaWVtZW5zLm5ldC9DTj1aWlpaWlpBNixMPVBLST9jZXJ0\naWZpY2F0ZVJldm9jYXRpb25MaXN0hklsZGFwOi8vY2wuc2llbWVucy5jb20vQ049\nWlpaWlpaQTYsbz1UcnVzdGNlbnRlcj9jZXJ0aWZpY2F0ZVJldm9jYXRpb25MaXN0\nMEUGA1UdIAQ+MDwwOgYNKwYBBAGhaQcCAgMBAzApMCcGCCsGAQUFBwIBFhtodHRw\nOi8vd3d3LnNpZW1lbnMuY29tL3BraS8wDAYDVR0TAQH/BAIwADAfBgNVHSMEGDAW\ngBT4FV1HDGx3e3LEAheRaKK292oJRDCCAQQGCCsGAQUFBwEBBIH3MIH0MDIGCCsG\nAQUFBzAChiZodHRwOi8vYWguc2llbWVucy5jb20vcGtpP1paWlpaWkE2LmNydDBB\nBggrBgEFBQcwAoY1bGRhcDovL2FsLnNpZW1lbnMubmV0L0NOPVpaWlpaWkE2LEw9\nUEtJP2NBQ2VydGlmaWNhdGUwSQYIKwYBBQUHMAKGPWxkYXA6Ly9hbC5zaWVtZW5z\nLmNvbS9DTj1aWlpaWlpBNixvPVRydXN0Y2VudGVyP2NBQ2VydGlmaWNhdGUwMAYI\nKwYBBQUHMAGGJGh0dHA6Ly9vY3NwLnBraS1zZXJ2aWNlcy5zaWVtZW5zLmNvbTAN\nBgkqhkiG9w0BAQsFAAOCAgEAXPVcX6vaEcszJqg5IemF9aFTlwTrX5ITNIpzcqG+\nkD5haOf2mZYLjl+MKtLC1XfmIsGCUZNb8bjP6QHQEI+2d6x/ZOqPq7Kd7PwVu6x6\nxZrkDjUyhUbUntT5+RBy++l3Wf6Cq6Kx+K8ambHBP/bu90/p2U8KfFAG3Kr2gI2q\nfZrnNMOxmJfZ3/sXxssgLkhbZ7hRa+MpLfQ6uFsSiat3vlawBBvTyHnoZ/7oRc8y\nqi6QzWcd76CPpMElYWibl+hJzKbBZUWvc71AzHR6i1QeZ6wubYz7vr+FF5Y7tnxB\nVz6omPC9XAg0F+Dla6Zlz3Awj5imCzVXa+9SjtnsidmJdLcKzTAKyDewewoxYOOJ\nj3cJU7VSjJPl+2fVmDBaQwcNcUcu/TPAKApkegqO7tRF9IPhjhW8QkRnkqMetO3D\nOXmAFVIsEI0Hvb2cdb7B6jSpjGUuhaFm9TCKhQtCk2p8JCDTuaENLm1x34rrJKbT\n2vzyYN0CZtSkUdgD4yQxK9VWXGEzexRisWb4AnZjD2NAquLPpXmw8N0UwFD7MSpC\ndpaX7FktdvZmMXsnGiAdtLSbBgLVWOD1gmJFDjrhNbI8NOaOaNk4jrfGqNh5lhGU\n4DnBT2U6Cie1anLmFH/oZooAEXR2o3Nu+1mNDJChnJp0ovs08aa3zZvBdcloOvfU\nqdowggh3MIIGX6ADAgECAgQtyi/nMA0GCSqGSIb3DQEBCwUAMIGZMQswCQYDVQQG\nEwJERTEPMA0GA1UECAwGQmF5ZXJuMREwDwYDVQQHDAhNdWVuY2hlbjEQMA4GA1UE\nCgwHU2llbWVuczERMA8GA1UEBRMIWlpaWlpaQTExHTAbBgNVBAsMFFNpZW1lbnMg\nVHJ1c3QgQ2VudGVyMSIwIAYDVQQDDBlTaWVtZW5zIFJvb3QgQ0EgVjMuMCAyMDE2\nMB4XDTE2MDcyMDEzNDYxMFoXDTIyMDcyMDEzNDYxMFowgbYxCzAJBgNVBAYTAkRF\nMQ8wDQYDVQQIDAZCYXllcm4xETAPBgNVBAcMCE11ZW5jaGVuMRAwDgYDVQQKDAdT\naWVtZW5zMREwDwYDVQQFEwhaWlpaWlpBNjEdMBsGA1UECwwUU2llbWVucyBUcnVz\ndCBDZW50ZXIxPzA9BgNVBAMMNlNpZW1lbnMgSXNzdWluZyBDQSBNZWRpdW0gU3Ry\nZW5ndGggQXV0aGVudGljYXRpb24gMjAxNjCCAiIwDQYJKoZIhvcNAQEBBQADggIP\nADCCAgoCggIBAL9UfK+JAZEqVMVvECdYF9IK4KSw34AqyNl3rYP5x03dtmKaNu+2\n0fQqNESA1NGzw3s6LmrKLh1cR991nB2cvKOXu7AvEGpSuxzIcOROd4NpvRx+Ej1p\nJIPeqf+ScmVK7lMSO8QL/QzjHOpGV3is9sG+ZIxOW9U1ESooy4Hal6ZNs4DNItsz\npiCKqm6G3et4r2WqCy2RRuSqvnmMza7Y8BZsLy0ZVo5teObQ37E/FxqSrbDI8nxn\nB7nVUve5ZjrqoIGSkEOtyo11003dVO1vmWB9A0WQGDqE/q3w178hGhKfxzRaqzyi\nSoADUYS2sD/CglGTUxVq6u0pGLLsCFjItcCWqW+T9fPYfJ2CEd5b3hvqdCn+pXjZ\n/gdX1XAcdUF5lRnGWifaYpT9n4s4adzX8q6oHSJxTppuAwLRKH6eXALbGQ1I9lGQ\nDSOipD/09xkEsPw6HOepmf2U3YxZK1VU2sHqugFJboeLcHMzp6E1n2ctlNG1GKE9\nFDHmdyFzDi0Nnxtf/GgVjnHF68hByEE1MYdJ4nJLuxoT9hyjYdRW9MpeNNxxZnmz\nW3zh7QxIqP0ZfIz6XVhzrI9uZiqwwojDiM5tEOUkQ7XyW6grNXe75yt6mTj89LlB\nH5fOW2RNmCy/jzBXDjgyskgK7kuCvUYTuRv8ITXbBY5axFA+CpxZqokpAgMBAAGj\nggKmMIICojCCAQUGCCsGAQUFBwEBBIH4MIH1MEEGCCsGAQUFBzAChjVsZGFwOi8v\nYWwuc2llbWVucy5uZXQvQ049WlpaWlpaQTEsTD1QS0k/Y0FDZXJ0aWZpY2F0ZTAy\nBggrBgEFBQcwAoYmaHR0cDovL2FoLnNpZW1lbnMuY29tL3BraT9aWlpaWlpBMS5j\ncnQwSgYIKwYBBQUHMAKGPmxkYXA6Ly9hbC5zaWVtZW5zLmNvbS91aWQ9WlpaWlpa\nQTEsbz1UcnVzdGNlbnRlcj9jQUNlcnRpZmljYXRlMDAGCCsGAQUFBzABhiRodHRw\nOi8vb2NzcC5wa2ktc2VydmljZXMuc2llbWVucy5jb20wHwYDVR0jBBgwFoAUcG2g\nUOyp0CxnnRkV/v0EczXD4tQwEgYDVR0TAQH/BAgwBgEB/wIBADBABgNVHSAEOTA3\nMDUGCCsGAQQBoWkHMCkwJwYIKwYBBQUHAgEWG2h0dHA6Ly93d3cuc2llbWVucy5j\nb20vcGtpLzCBxwYDVR0fBIG/MIG8MIG5oIG2oIGzhj9sZGFwOi8vY2wuc2llbWVu\ncy5uZXQvQ049WlpaWlpaQTEsTD1QS0k/YXV0aG9yaXR5UmV2b2NhdGlvbkxpc3SG\nJmh0dHA6Ly9jaC5zaWVtZW5zLmNvbS9wa2k/WlpaWlpaQTEuY3JshkhsZGFwOi8v\nY2wuc2llbWVucy5jb20vdWlkPVpaWlpaWkExLG89VHJ1c3RjZW50ZXI/YXV0aG9y\naXR5UmV2b2NhdGlvbkxpc3QwJwYDVR0lBCAwHgYIKwYBBQUHAwIGCCsGAQUFBwME\nBggrBgEFBQcDCTAOBgNVHQ8BAf8EBAMCAQYwHQYDVR0OBBYEFPgVXUcMbHd7csQC\nF5Foorb3aglEMA0GCSqGSIb3DQEBCwUAA4ICAQBw+sqMp3SS7DVKcILEmXbdRAg3\nlLO1r457KY+YgCT9uX4VG5EdRKcGfWXK6VHGCi4Dos5eXFV34Mq/p8nu1sqMuoGP\nYjHn604eWDprhGy6GrTYdxzcE/GGHkpkuE3Ir/45UcmZlOU41SJ9SNjuIVrSHMOf\nccSY42BCspR/Q1Z/ykmIqQecdT3/Kkx02GzzSN2+HlW6cEO4GBW5RMqsvd2n0h2d\nfe2zcqOgkLtx7u2JCR/U77zfyxG3qXtcymoz0wgSHcsKIl+GUjITLkHfS9Op8V7C\nGr/dX437sIg5pVHmEAWadjkIzqdHux+EF94Z6kaHywohc1xG0KvPYPX7iSNjkvhz\n4NY53DHmxl4YEMLffZnaS/dqyhe1GTpcpyN8WiR4KuPfxrkVDOsuzWFtMSvNdlOV\ngdI0MXcLMP+EOeANZWX6lGgJ3vWyemo58nzgshKd24MY3w3i6masUkxJH2KvI7UH\n/1Db3SC8oOUjInvSRej6M3ZhYWgugm6gbpUgFoDw/o9Cg6Qm71hY0JtcaPC13rzm\nN8a2Br0+Fa5e2VhwLmAxyfe1JKzqPwuHT0S5u05SQghL5VdzqfA8FCL/j4XC9yI6\ncsZTAQi73xFQYVjZt3+aoSz84lOlTmVo/jgvGMY/JzH9I4mETGgAJRNj34Z/0meh\nM+pKWCojNH/dgyJSwDGCAlIwggJOAgEBMIG/MIG2MQswCQYDVQQGEwJERTEPMA0G\nA1UECAwGQmF5ZXJuMREwDwYDVQQHDAhNdWVuY2hlbjEQMA4GA1UECgwHU2llbWVu\nczERMA8GA1UEBRMIWlpaWlpaQTYxHTAbBgNVBAsMFFNpZW1lbnMgVHJ1c3QgQ2Vu\ndGVyMT8wPQYDVQQDDDZTaWVtZW5zIElzc3VpbmcgQ0EgTWVkaXVtIFN0cmVuZ3Ro\nIEF1dGhlbnRpY2F0aW9uIDIwMTYCBBXXLOIwCwYJYIZIAWUDBAIBoGkwHAYJKoZI\nhvcNAQkFMQ8XDTE5MTEyMDE0NTYyMFowLwYJKoZIhvcNAQkEMSIEIJDnZUpcVLzC\nOdtpkH8gtxwLPIDE0NmAmFC9uM8q2z+OMBgGCSqGSIb3DQEJAzELBgkqhkiG9w0B\nBwEwCwYJKoZIhvcNAQEBBIIBAH/Pqv2xp3a0jSPkwU1K3eGA/1lfoNJMUny4d/PS\nLVWlkgrmedXdLmuBzAGEaaZOJS0lEpNd01pR/reHs7xxZ+RZ0olTs2ufM0CijQSx\nOL9HDl2O3OoD77NWx4tl3Wy1yJCeV3XH/cEI7AkKHCmKY9QMoMYWh16ORBtr+YcS\nYK+gONOjpjgcgTJgZ3HSFgQ50xiD4WT1kFBHsuYsLqaOSbTfTN6Ayyg4edjrPQqa\nVcVf1OQcIrfWA3yMQrnEZfOYfN/D4EPjTfxBV+VCi/F2bdZmMbJ7jNk1FbewSwWO\nSDH1i0K32NyFbnh0BSos7njq7ELqKlYBsoB/sZfaH2vKy5U=\n-----END SIGNED MESSAGE-----"),
			MessageSize:  6494,
			Tagger: &gitalypb.CommitAuthor{
				Name:     []byte("Roger Meier"),
				Email:    []byte("r.meier@siemens.com"),
				Date:     &timestamppb.Timestamp{Seconds: 1574261780},
				Timezone: []byte("+0100"),
			},
			SignatureType: gitalypb.SignatureType_X509,
		},
		{
			Name:        []byte("v1.2.0"),
			Id:          annotatedTagID.String(),
			Message:     []byte("Blob tag"),
			MessageSize: 8,
			Tagger: &gitalypb.CommitAuthor{
				Name:     []byte("Scrooge McDuck"),
				Email:    []byte("scrooge@mcduck.com"),
				Date:     &timestamppb.Timestamp{Seconds: 1572776879},
				Timezone: []byte("+0100"),
			},
		},
		{
			Name:         []byte("v1.3.0"),
			Id:           commitID.String(),
			TargetCommit: gitCommit,
		},
		{
			Name: []byte("v1.4.0"),
			Id:   blobID.String(),
		},
		{
			Name:         []byte("v1.5.0"),
			Id:           commitID.String(),
			TargetCommit: gitCommit,
		},
		{
			Name:         []byte("v1.6.0"),
			Id:           bigCommitID.String(),
			TargetCommit: bigCommit,
		},
		{
			Name:         []byte("v1.7.0"),
			Id:           bigMessageTag1ID.String(),
			Message:      []byte(bigMessage[:helper.MaxCommitOrTagMessageSize]),
			MessageSize:  int64(len(bigMessage)),
			TargetCommit: gitCommit,
			Tagger: &gitalypb.CommitAuthor{
				Name:     []byte("Scrooge McDuck"),
				Email:    []byte("scrooge@mcduck.com"),
				Date:     &timestamppb.Timestamp{Seconds: 1572776879},
				Timezone: []byte("+0100"),
			},
		},
	}

	for _, expectedTag := range expectedTags {
		rpcRequest := &gitalypb.FindTagRequest{Repository: repoProto, TagName: expectedTag.Name}

		resp, err := client.FindTag(ctx, rpcRequest)
		require.NoError(t, err)

		testhelper.ProtoEqual(t, expectedTag, resp.GetTag())
	}
}

func TestFindTagNestedTag(t *testing.T) {
	cfg, client := setupRefServiceWithoutRepo(t)

	repoProto, repoPath := gittest.CloneRepo(t, cfg, cfg.Storages[0])
	repo := localrepo.NewTestRepo(t, cfg, repoProto)

	blobID := git.ObjectID("faaf198af3a36dbf41961466703cc1d47c61d051")
	commitID := git.ObjectID("6f6d7e7ed97bb5f0054f2b1df789b39ca89b6ff9")

	ctx, cancel := testhelper.Context()
	defer cancel()

	testCases := []struct {
		description string
		depth       int
		originalOid git.ObjectID
	}{
		{
			description: "nested 1 deep, points to a commit",
			depth:       1,
			originalOid: commitID,
		},
		{
			description: "nested 4 deep, points to a commit",
			depth:       4,
			originalOid: commitID,
		},
		{
			description: "nested 3 deep, points to a blob",
			depth:       3,
			originalOid: blobID,
		},
		{
			description: "nested 20 deep, points to a commit",
			depth:       20,
			originalOid: commitID,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tags, err := repo.GetReferences(ctx, "refs/tags/")
			require.NoError(t, err)

			updater, err := updateref.New(ctx, cfg, repo)
			require.NoError(t, err)
			for _, tag := range tags {
				require.NoError(t, updater.Delete(tag.Name))
			}
			require.NoError(t, updater.Commit())

			catfileCache := catfile.NewCache(cfg)
			defer catfileCache.Stop()

			objectReader, err := catfileCache.ObjectReader(ctx, repo)
			require.NoError(t, err)

			objectInfoReader, err := catfileCache.ObjectInfoReader(ctx, repo)
			require.NoError(t, err)

			info, err := objectInfoReader.Info(ctx, git.Revision(tc.originalOid))
			require.NoError(t, err)

			tagID := tc.originalOid
			var tagName, tagMessage string

			for depth := 0; depth < tc.depth; depth++ {
				tagName = fmt.Sprintf("tag-depth-%d", depth)
				tagMessage = fmt.Sprintf("a commit %d deep", depth)
				tagID = gittest.WriteTag(t, cfg, repoPath, tagName, tagID.Revision(), gittest.WriteTagConfig{Message: tagMessage})
			}
			expectedTag := &gitalypb.Tag{
				Name:        []byte(tagName),
				Id:          tagID.String(),
				Message:     []byte(tagMessage),
				MessageSize: int64(len([]byte(tagMessage))),
				Tagger: &gitalypb.CommitAuthor{
					Name:     []byte("Scrooge McDuck"),
					Email:    []byte("scrooge@mcduck.com"),
					Date:     &timestamppb.Timestamp{Seconds: 1572776879},
					Timezone: []byte("+0100"),
				},
			}
			if info.Type == "commit" {
				commit, err := catfile.GetCommit(ctx, objectReader, git.Revision(tc.originalOid))
				require.NoError(t, err)
				expectedTag.TargetCommit = commit
			}
			rpcRequest := &gitalypb.FindTagRequest{Repository: repoProto, TagName: []byte(tagName)}

			resp, err := client.FindTag(ctx, rpcRequest)
			require.NoError(t, err)
			require.Equal(t, expectedTag, resp.GetTag())
		})
	}
}

func TestInvalidFindTagRequest(t *testing.T) {
	_, repo, _, client := setupRefService(t)

	testCases := []struct {
		desc    string
		request *gitalypb.FindTagRequest
	}{
		{
			desc:    "empty request",
			request: &gitalypb.FindTagRequest{},
		},
		{
			desc: "invalid repo",
			request: &gitalypb.FindTagRequest{
				Repository: &gitalypb.Repository{
					StorageName:  "fake",
					RelativePath: "repo",
				},
			},
		},
		{
			desc: "empty tag name",
			request: &gitalypb.FindTagRequest{
				Repository: repo,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()
			_, err := client.FindTag(ctx, tc.request)
			testhelper.RequireGrpcCode(t, err, codes.InvalidArgument)
		})
	}
}
