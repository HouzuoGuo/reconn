package httpsvc

import (
	"context"

	"github.com/HouzuoGuo/reconn-voice-clone/shared"
)

func (svc *HttpService) DownloadBlobToLocalFileIfNotExist(ctx context.Context, blobContainerName, fileName, localDir string) (string, error) {
	return shared.DownloadBlobToLocalFileIfNotExist(ctx, svc.BlobClient, blobContainerName, fileName, localDir)
}

func (svc *HttpService) DownloadModelIfNotExist(ctx context.Context, fileName string) (string, error) {
	return svc.DownloadBlobToLocalFileIfNotExist(ctx, svc.Config.VoiceModelContainer, fileName, svc.Config.VoiceModelDir)
}

func (svc *HttpService) UploadFromLocalFile(ctx context.Context, blobContainerName, fileName, localDir string) error {
	return shared.UploadFromLocalFile(ctx, svc.BlobClient, blobContainerName, fileName, localDir)
}

func (svc *HttpService) UploadAndSave(ctx context.Context, blobContainerName, fileName, localDir string, data []byte) (string, error) {
	return shared.UploadAndSave(ctx, svc.BlobClient, blobContainerName, fileName, localDir, data)
}
