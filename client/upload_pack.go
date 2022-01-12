package client

import (
	"context"
	"fmt"
	"io"

	"gitlab.com/gitlab-org/gitaly/v14/internal/git/pktline"
	"gitlab.com/gitlab-org/gitaly/v14/internal/stream"
	"gitlab.com/gitlab-org/gitaly/v14/proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/v14/streamio"
	"google.golang.org/grpc"
)

// UploadPack proxies an SSH git-upload-pack (git fetch) session to Gitaly
func UploadPack(ctx context.Context, conn *grpc.ClientConn, stdin io.Reader, stdout, stderr io.Writer, req *gitalypb.SSHUploadPackRequest) (int32, error) {
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	ssh := gitalypb.NewSSHServiceClient(conn)
	uploadPackStream, err := ssh.SSHUploadPack(ctx2)
	if err != nil {
		return 0, err
	}

	if err = uploadPackStream.Send(req); err != nil {
		return 0, err
	}

	inWriter := streamio.NewWriter(func(p []byte) error {
		return uploadPackStream.Send(&gitalypb.SSHUploadPackRequest{Stdin: p})
	})

	return stream.Handler(func() (stream.StdoutStderrResponse, error) {
		return uploadPackStream.Recv()
	}, func(errC chan error) {
		_, errRecv := io.Copy(inWriter, stdin)
		if err := uploadPackStream.CloseSend(); err != nil && errRecv == nil {
			errC <- err
		} else {
			errC <- errRecv
		}
	}, stdout, stderr)
}

// UploadPackWithSidechannel proxies an SSH git-upload-pack (git fetch)
// session to Gitaly using a sidechannel for the raw data transfer.
func UploadPackWithSidechannel(ctx context.Context, conn *grpc.ClientConn, reg *SidechannelRegistry, stdin io.Reader, stdout, stderr io.Writer, req *gitalypb.SSHUploadPackWithSidechannelRequest) (int32, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	copyRequest := func(c SidechannelConn) error {
		if _, err := io.Copy(c, stdin); err != nil {
			return fmt.Errorf("copy request: %w", err)
		}
		if err := c.CloseWrite(); err != nil {
			return fmt.Errorf("close request: %w", err)
		}
		return nil
	}

	copyResponse := func(c SidechannelConn) error {
		if err := pktline.EachSidebandPacket(c, func(band byte, data []byte) (err error) {
			switch band {
			case 1:
				_, err = stdout.Write(data)
			case 2:
				_, err = stderr.Write(data)
			default:
				err = fmt.Errorf("unexpected band: %d", band)
			}
			return err
		}); err != nil {
			return fmt.Errorf("copy response: %w", err)
		}

		return nil
	}

	ctx, wt := reg.Register(ctx, func(c SidechannelConn) error {
		request := make(chan error, 1)
		go func() { request <- copyRequest(c) }()

		response := make(chan error, 1)
		go func() { response <- copyResponse(c) }()

		for {
			select {
			case err := <-response:
				// Server is done. No point in waiting for client.
				return err
			case err := <-request:
				// Client is done; wait for server to finish too.
				if err != nil {
					return err
				}
			}
		}
	})
	defer wt.Close()

	sshClient := gitalypb.NewSSHServiceClient(conn)
	if _, err := sshClient.SSHUploadPackWithSidechannel(ctx, req); err != nil {
		return 0, err
	}

	if err := wt.Close(); err != nil {
		return 0, err
	}

	return 0, nil
}
