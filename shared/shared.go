package shared

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// GPUTask describes the paramters of a GPU task intended for the GPU-enabled workers.
type GPUTask struct {
	// VoiceModelID is the voice model ID record in database for the GPU worker to create a voice model.
	VoiceModelID int

	// AIReplyPersonID is the AI person ID in database for the GPU worker to perform TTS.
	AIReplyPersonID int
	// AIReplyPersonID is the AI person reply ID in database for the GPU worker to perform TTS.
	AIReplyVoiceID int
}

// CloneRealTimeResponse is the structure of /clone-rt/ response.
type CloneRealTimeResponse struct {
	// ModelDestinationFile is the relative path of the newly cloned voice model.
	ModelDestinationFile string `json:"model"`
}

// TextToSpeechRealTimeRequest is the structure of /tts-rt/ request.
type TextToSpeechRealTimeRequest struct {
	Text         string  `json:"text"`
	TopK         float64 `json:"topK"`
	TopP         float64 `json:"topP"`
	MineosP      float64 `json:"mineosP"`
	SemanticTemp float64 `json:"semanticTemp"`
	WaveformTemp float64 `json:"waveformTemp"`
	FineTemp     float64 `json:"fineTemp"`
}

func DownloadBlobToLocalFileIfNotExist(ctx context.Context, blobClient *azblob.Client, blobContainerName, fileName, localDir string) (string, error) {
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
	_, err = blobClient.DownloadFile(ctx, blobContainerName, fileName, localFile, nil)
	return localFilePath, err
}

func UploadFromLocalFile(ctx context.Context, blobClient *azblob.Client, blobContainerName, fileName, localDir string) error {
	if localDirStat, err := os.Stat(localDir); err != nil || !localDirStat.IsDir() {
		err = fmt.Errorf("cannot access local fs directory %q: %w", localDir, err)
		return err
	}
	localFile, err := os.Open(path.Join(localDir, fileName))
	if err != nil {
		return err
	}
	defer localFile.Close()
	_, err = blobClient.UploadFile(ctx, blobContainerName, fileName, localFile, nil)
	if err != nil {
		return err
	}
	return nil
}

func UploadAndSave(ctx context.Context, blobClient *azblob.Client, blobContainerName, fileName, localDir string, data []byte) (string, error) {
	if localDirStat, err := os.Stat(localDir); err != nil || !localDirStat.IsDir() {
		err = fmt.Errorf("cannot access local fs directory %q: %w", localDir, err)
		return "", err
	}
	localFilePath := path.Join(localDir, fileName)
	if err := os.WriteFile(localFilePath, data, 0644); err != nil {
		return "", err
	}
	if _, err := blobClient.UploadBuffer(ctx, blobContainerName, fileName, data, nil); err != nil {
		return "", err
	}
	return localFilePath, nil
}
