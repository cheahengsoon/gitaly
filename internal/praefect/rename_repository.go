package praefect

import (
	"errors"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/v14/internal/gitaly/storage"
	"gitlab.com/gitlab-org/gitaly/v14/internal/helper"
	"gitlab.com/gitlab-org/gitaly/v14/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/v14/internal/praefect/commonerr"
	"gitlab.com/gitlab-org/gitaly/v14/internal/praefect/datastore"
	"gitlab.com/gitlab-org/gitaly/v14/proto/go/gitalypb"
	"google.golang.org/grpc"
)

// RenameRepositoryHandler handles /gitaly.RepositoryService/RenameRepository calls by renaming
// the repository in the lookup table stored in the database.
func RenameRepositoryHandler(rs datastore.RepositoryStore) grpc.StreamHandler {
	return func(srv interface{}, stream grpc.ServerStream) error {
		var req gitalypb.RenameRepositoryRequest
		if err := stream.RecvMsg(&req); err != nil {
			return fmt.Errorf("receive request: %w", err)
		}

		// These checks are not strictly necessary but they exist to keep retain compatibility with
		// Gitaly's tested behavior.
		if req.GetRepository() == nil {
			return helper.ErrInvalidArgumentf("missing repository")
		} else if req.GetRelativePath() == "" {
			return helper.ErrInvalidArgumentf("missing new relativePath")
		} else if _, err := storage.ValidateRelativePath("/root", req.GetRelativePath()); err != nil {
			return helper.ErrInvalidArgument(err)
		}

		if err := rs.RenameRepositoryInPlace(stream.Context(),
			req.GetRepository().GetStorageName(),
			req.GetRepository().GetRelativePath(),
			req.GetRelativePath(),
		); err != nil {
			if errors.Is(err, commonerr.ErrRepositoryNotFound) {
				return helper.ErrNotFound(err)
			} else if errors.Is(err, commonerr.ErrRepositoryAlreadyExists) {
				const msg = "destination already exists"
				if featureflag.RenameRepositoryLocking.IsEnabled(stream.Context()) {
					return helper.ErrAlreadyExistsf(msg)
				}

				return helper.ErrFailedPreconditionf(msg)
			}

			return helper.ErrInternal(err)
		}

		return stream.SendMsg(&gitalypb.RenameRepositoryResponse{})
	}
}
