package shredder

import (
    "crypto/rand"
    "io"
    "github.com/spf13/afero"
    "os"
)

// Use the real file system by default
var AppFs = afero.NewOsFs()
var ShredOverwriteCount = 3

func OverwriteStreamWithRandomBytes(writer io.Writer, length int64) {
    for i := 0; i < ShredOverwriteCount; i++ {
        randomBytes := GenerateRandomBytes(length)

        // Write the random bytes to the stream
        _, err := writer.Write(randomBytes)

        if err != nil {
            panic("Error writing random bytes to stream: " + err.Error())
        }

        // Sync the writer to ensure the data is written
        // If it doesn't support sync, that's fine
        if syncer, ok := writer.(interface {
            Sync() error
        }); ok {
            syncErr := syncer.Sync()
            if syncErr != nil {
                panic("Error syncing writer: " + syncErr.Error())
            }
        }

        // Seek to the beginning of the stream - we do need
        // to do this, so panic if it's not supported
        if seeker, ok := writer.(interface {
            Seek(offset int64, whence int) (int64, error)
        }); ok {
            _, seekErr := seeker.Seek(0, 0)
            if seekErr != nil {
                panic("Error seeking writer: " + seekErr.Error())
            } 
        } else {
            panic("Writer does not support seeking")
        }
    }
}

func GenerateRandomBytes(length int64) []byte {
    randomBytes := make([]byte, length)

    _, err := rand.Read(randomBytes)

    if err != nil {
        panic("Error generating random bytes: " + err.Error())
    }

    return randomBytes
}

func GetFileLength(file afero.File) int64 {
    fileInfo, err := file.Stat()
    if err != nil {
        panic("Error getting file statistics: " + err.Error())
    }

    return fileInfo.Size()
}

func Shred(pathToFile string) {
    file, err := AppFs.OpenFile(pathToFile, os.O_RDWR, 0644)

    if err != nil {
        panic("Error opening file: " + err.Error())
    }

    // Now we know the file exists and is open, we can defer
    // the close and make sure it gets closed regardless of errors
    defer file.Close()

    fileLength := GetFileLength(file)
    fileStream := io.Writer(file)
    OverwriteStreamWithRandomBytes(fileStream, fileLength)
}


