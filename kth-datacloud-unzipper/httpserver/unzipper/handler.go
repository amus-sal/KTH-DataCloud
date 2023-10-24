package unzipper

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:generate moq -out handler_mocks.go . encoder stripeService

type encoder interface {
	Respond(ctx context.Context, w http.ResponseWriter, payload interface{}, statusCode int)
	Error(ctx context.Context, w http.ResponseWriter, err error)
}

type Handler struct {
	encoder   encoder
	publisher publisher
}
type publisher interface {
	FileCreated(ctx context.Context, eventID string, filePath string) error
}

// NewHandler returns a new Stripe handler with all dependencies.
func NewHandler(encoder encoder, publisher publisher) Handler {
	return Handler{
		encoder:   encoder,
		publisher: publisher,
	}
}

func (h Handler) Routes(r chi.Router) {
	r.Post("/", h.postEvent)
}

func (h Handler) postEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// handle

	err := unzip("./file.zip", "")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("received request")

	h.publisher.FileCreated(ctx, "123", "/file.tsv")
	// Empty response to Stripe with http 200 to acknowledge we handled their request.
	h.encoder.Respond(ctx, w, struct{}{}, http.StatusOK)
}

func unzip(source, destination string) error {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	// 2. Get the absolute destination path
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}

	// 3. Iterate over zip files inside the archive and unzip each of them
	for _, f := range reader.File {
		err := unzipFile(f, destination)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipFile(f *zip.File, destination string) error {
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// 6. Create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil
}
