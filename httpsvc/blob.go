package httpsvc

import (
	"context"
	"fmt"
	"os"
	"path"
)

func (svc *HttpService) DownloadBlobToLocalFileIfNotExist(ctx context.Context, blobContainerName, fileName, localDir string) (string, error) {
	localFilePath := path.Join(localDir, fileName)
	if localDirStat, err := os.Stat(localDir); err != nil || !localDirStat.IsDir() {
		err = fmt.Errorf("cannot access local fs directory %q: %w", localDir, err)
		return "", err
	}
	if stat, err := os.Stat(localFilePath); err == nil && stat.Size() > 0 {
		// Already downloaded to disk.
		return localFilePath, nil
	}
	localFile, err := os.Create(localFilePath)
	defer localFile.Close()
	_, err = svc.Config.BlobClient.DownloadFile(ctx, blobContainerName, fileName, localFile, nil)
	return localFilePath, err
}

func (svc *HttpService) DownloadModelIfNotExist(ctx context.Context, fileName string) (string, error) {
	return svc.DownloadBlobToLocalFileIfNotExist(ctx, svc.Config.VoiceModelContainer, fileName, svc.Config.VoiceModelDir)
}

func (svc *HttpService) UploadFromLocalFile(ctx context.Context, blobContainerName, fileName, localDir string) error {
	if localDirStat, err := os.Stat(localDir); err != nil || !localDirStat.IsDir() {
		err = fmt.Errorf("cannot access local fs directory %q: %w", localDir, err)
		return err
	}
	localFile, err := os.Open(path.Join(localDir, fileName))
	if err != nil {
		return err
	}
	defer localFile.Close()
	_, err = svc.Config.BlobClient.UploadFile(ctx, blobContainerName, fileName, localFile, nil)
	if err != nil {
		return err
	}
	return nil
}

func (svc *HttpService) UploadAndSave(ctx context.Context, blobContainerName, fileName, localDir string, data []byte) (string, error) {
	if localDirStat, err := os.Stat(localDir); err != nil || !localDirStat.IsDir() {
		err = fmt.Errorf("cannot access local fs directory %q: %w", localDir, err)
		return "", err
	}
	localFilePath := path.Join(localDir, fileName)
	if err := os.WriteFile(localFilePath, data, 0644); err != nil {
		return "", err
	}
	if _, err := svc.Config.BlobClient.UploadBuffer(ctx, blobContainerName, fileName, data, nil); err != nil {
		return "", err
	}
	return localFilePath, nil
}
