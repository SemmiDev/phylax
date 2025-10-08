package domain

type Compressor interface {
	Compress(sourcePath, destPath string) error
	Decompress(sourcePath, destPath string) error
}
