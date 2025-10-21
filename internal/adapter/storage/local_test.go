package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLocalStorage(t *testing.T) {
	Convey("Given a LocalStorage", t, func() {
		tempDir, err := os.MkdirTemp("", "local_storage_test")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)

		Convey("NewLocal", func() {
			Convey("When creating with valid path", func() {
				storage, err := NewLocal(tempDir)

				Convey("It should create successfully", func() {
					So(err, ShouldBeNil)
					So(storage, ShouldNotBeNil)
					So(storage.basePath, ShouldEqual, tempDir)
				})
			})

			Convey("When creating with non-existent path", func() {
				newPath := filepath.Join(tempDir, "new", "nested", "dir")
				storage, err := NewLocal(newPath)

				Convey("It should create directory and succeed", func() {
					So(err, ShouldBeNil)
					So(storage, ShouldNotBeNil)

					// Verify directory exists
					info, err := os.Stat(newPath)
					So(err, ShouldBeNil)
					So(info.IsDir(), ShouldBeTrue)
				})
			})
		})

		Convey("Upload method", func() {
			storage, _ := NewLocal(tempDir)

			Convey("When uploading a valid file", func() {
				// Create source file
				sourceFile := filepath.Join(tempDir, "source.txt")
				os.WriteFile(sourceFile, []byte("test content"), 0644)

				ctx := context.Background()
				err := storage.Upload(ctx, sourceFile, "uploaded.txt")

				Convey("It should upload successfully", func() {
					So(err, ShouldBeNil)

					// Verify file exists
					uploadedPath := filepath.Join(tempDir, "uploaded.txt")
					content, err := os.ReadFile(uploadedPath)
					So(err, ShouldBeNil)
					So(string(content), ShouldEqual, "test content")
				})
			})

			Convey("When source file does not exist", func() {
				ctx := context.Background()
				err := storage.Upload(ctx, "nonexistent.txt", "uploaded.txt")

				Convey("It should return error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "failed to open source")
				})
			})
		})

		Convey("List method", func() {
			storage, _ := NewLocal(tempDir)

			Convey("When directory has files", func() {
				// Create test files
				os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test"), 0644)
				os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("test"), 0644)
				os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)

				ctx := context.Background()
				files, err := storage.List(ctx)

				Convey("It should list only files", func() {
					So(err, ShouldBeNil)
					So(len(files), ShouldEqual, 2)
					So(files, ShouldContain, "file1.txt")
					So(files, ShouldContain, "file2.txt")
					So(files, ShouldNotContain, "subdir")
				})
			})

			Convey("When directory is empty", func() {
				emptyDir := filepath.Join(tempDir, "empty")
				os.Mkdir(emptyDir, 0755)
				storage, _ := NewLocal(emptyDir)

				ctx := context.Background()
				files, err := storage.List(ctx)

				Convey("It should return empty list", func() {
					So(err, ShouldBeNil)
					So(len(files), ShouldEqual, 0)
				})
			})
		})

		Convey("Delete method", func() {
			storage, _ := NewLocal(tempDir)

			Convey("When deleting existing file", func() {
				// Create test file
				testFile := "delete_me.txt"
				os.WriteFile(filepath.Join(tempDir, testFile), []byte("test"), 0644)

				ctx := context.Background()
				err := storage.Delete(ctx, testFile)

				Convey("It should delete successfully", func() {
					So(err, ShouldBeNil)

					// Verify file is deleted
					_, err := os.Stat(filepath.Join(tempDir, testFile))
					So(os.IsNotExist(err), ShouldBeTrue)
				})
			})

			Convey("When deleting non-existent file", func() {
				ctx := context.Background()
				err := storage.Delete(ctx, "nonexistent.txt")

				Convey("It should return error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "failed to delete file")
				})
			})
		})

		Convey("GetOldFiles method", func() {
			storage, _ := NewLocal(tempDir)

			Convey("When finding old files", func() {
				// Create old file
				oldFile := filepath.Join(tempDir, "old.txt")
				os.WriteFile(oldFile, []byte("test"), 0644)
				oldTime := time.Now().Add(-10 * 24 * time.Hour)
				os.Chtimes(oldFile, oldTime, oldTime)

				// Create new file
				newFile := filepath.Join(tempDir, "new.txt")
				os.WriteFile(newFile, []byte("test"), 0644)

				ctx := context.Background()
				cutoff := time.Now().Add(-7 * 24 * time.Hour)
				oldFiles, err := storage.GetOldFiles(ctx, cutoff)

				Convey("It should return only old files", func() {
					So(err, ShouldBeNil)
					So(len(oldFiles), ShouldEqual, 1)
					So(oldFiles[0], ShouldEqual, "old.txt")
				})
			})
		})

		Convey("GetPath method", func() {
			storage, _ := NewLocal(tempDir)

			Convey("When getting path for filename", func() {
				filename := "test.txt"
				path := storage.GetPath(filename)

				Convey("It should return full path", func() {
					expected := filepath.Join(tempDir, filename)
					So(path, ShouldEqual, expected)
				})
			})
		})
	})
}
