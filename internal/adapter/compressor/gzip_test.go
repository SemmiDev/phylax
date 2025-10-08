package compressor

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGzipCompressor(t *testing.T) {
	Convey("Given a GzipCompressor", t, func() {
		compressor := NewGzip()

		Convey("Compress method", func() {
			Convey("When compressing a valid file", func() {
				// Create a temporary input file with some content
				inputContent := []byte("This is a test content for compression")
				inputFile, err := os.CreateTemp("", "test_input_*.txt")
				So(err, ShouldBeNil)
				defer os.Remove(inputFile.Name())

				_, err = inputFile.Write(inputContent)
				So(err, ShouldBeNil)
				inputFile.Close()

				// Create a temporary output file path
				outputFile := filepath.Join(os.TempDir(), "test_output.gz")

				Convey("It should compress successfully", func() {
					err := compressor.Compress(inputFile.Name(), outputFile)
					So(err, ShouldBeNil)

					// Verify the output file exists and is a valid gzip file
					_, err = os.Stat(outputFile)
					So(err, ShouldBeNil)

					gzipFile, err := os.Open(outputFile)
					So(err, ShouldBeNil)
					defer gzipFile.Close()

					gzipReader, err := gzip.NewReader(gzipFile)
					So(err, ShouldBeNil)
					defer gzipReader.Close()

					// Read the compressed content
					var decompressedContent bytes.Buffer
					_, err = decompressedContent.ReadFrom(gzipReader)
					So(err, ShouldBeNil)
					So(decompressedContent.Bytes(), ShouldResemble, inputContent)

					os.Remove(outputFile)
				})
			})

			Convey("When the source file does not exist", func() {
				err := compressor.Compress("nonexistent.txt", "output.gz")
				Convey("It should return an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "failed to open source file")
				})
			})

			Convey("When the destination path is invalid", func() {
				inputFile, err := os.CreateTemp("", "test_input_*.txt")
				So(err, ShouldBeNil)
				defer os.Remove(inputFile.Name())

				err = compressor.Compress(inputFile.Name(), "/invalid/path/output.gz")
				Convey("It should return an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "failed to create dest file")
				})
			})
		})

		Convey("Decompress method", func() {
			Convey("When decompressing a valid gzip file", func() {
				// Create a temporary gzip file
				inputContent := []byte("This is a test content for decompression")
				gzipFile, err := os.CreateTemp("", "test_input_*.gz")
				So(err, ShouldBeNil)
				defer os.Remove(gzipFile.Name())

				gzipWriter, err := gzip.NewWriterLevel(gzipFile, gzip.BestCompression)
				So(err, ShouldBeNil)
				_, err = gzipWriter.Write(inputContent)
				So(err, ShouldBeNil)
				gzipWriter.Close()
				gzipFile.Close()

				// Create a temporary output file path
				outputFile := filepath.Join(os.TempDir(), "test_output.txt")

				Convey("It should decompress successfully", func() {
					err := compressor.Decompress(gzipFile.Name(), outputFile)
					So(err, ShouldBeNil)

					// Verify the output file exists and contains the correct content
					decompressedContent, err := os.ReadFile(outputFile)
					So(err, ShouldBeNil)
					So(decompressedContent, ShouldResemble, inputContent)

					os.Remove(outputFile)
				})
			})

			Convey("When the source file does not exist", func() {
				err := compressor.Decompress("nonexistent.gz", "output.txt")
				Convey("It should return an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "failed to open source file")
				})
			})

			Convey("When the source file is not a valid gzip file", func() {
				// Create a temporary non-gzip file
				invalidFile, err := os.CreateTemp("", "test_invalid_*.txt")
				So(err, ShouldBeNil)
				defer os.Remove(invalidFile.Name())

				_, err = invalidFile.Write([]byte("not a gzip file"))
				So(err, ShouldBeNil)
				invalidFile.Close()

				outputFile := filepath.Join(os.TempDir(), "test_output.txt")

				err = compressor.Decompress(invalidFile.Name(), outputFile)
				Convey("It should return an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "failed to create gzip reader")
				})
			})

			Convey("When the destination path is invalid", func() {
				// Create a temporary valid gzip file
				gzipFile, err := os.CreateTemp("", "test_input_*.gz")
				So(err, ShouldBeNil)
				defer os.Remove(gzipFile.Name())

				gzipWriter, err := gzip.NewWriterLevel(gzipFile, gzip.BestCompression)
				So(err, ShouldBeNil)
				_, err = gzipWriter.Write([]byte("test content"))
				So(err, ShouldBeNil)
				gzipWriter.Close()
				gzipFile.Close()

				err = compressor.Decompress(gzipFile.Name(), "/invalid/path/output.txt")
				Convey("It should return an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "failed to create dest file")
				})
			})
		})
	})
}
