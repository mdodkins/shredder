package shredder

import (
    "testing"
    "bytes"
    "reflect"
    "errors"
    "crypto/rand"
    "github.com/spf13/afero"
    "os"
)

func TestGenerateRandomBytes(t *testing.T) {
    expected := 8
    actual := len(GenerateRandomBytes(8))

    if actual != expected {
        t.Errorf("Test failed, expected: '%d', got:  '%d'", expected, actual)
    }
}

// Write error testing
type WriterThatErrorsOnWrite struct {
    buf *bytes.Buffer
}

func (w *WriterThatErrorsOnWrite) Write(p []byte) (n int, err error) {
    return 0, errors.New("Some awful write error")
}

func TestPanicWhenWriterErrors(t *testing.T) {
    // Given
    var arbitraryLength int64 = 6
    writer := &WriterThatErrorsOnWrite{} 

    // Then
    defer func() {
        if r := recover(); r == nil {
            t.Errorf("The code did not panic")
        }
    }()

    // When
    OverwriteStreamWithRandomBytes(writer, arbitraryLength)
}

// Sync error testing
type WriterThatErrorsOnSync struct {
    buf *bytes.Buffer
}

func (w *WriterThatErrorsOnSync) Write(p []byte) (n int, err error) {
    return w.buf.Write(p)
}

func (w *WriterThatErrorsOnSync) Sync() error {
    return errors.New("Some awful sync error")
}

func TestPanicWhenWriterErrorsOnSync(t *testing.T) {
    // Given
    writer := &WriterThatErrorsOnSync{buf: &bytes.Buffer{}}
    var arbitraryLength int64 = 6
    // Then
    defer func() {
        if r := recover(); r == nil {
            t.Errorf("The code did not panic")
        }
    }()

    // When
    OverwriteStreamWithRandomBytes(writer, arbitraryLength)
}

type WriterThatDoesNotImplementSync struct {
    buf *bytes.Buffer
}

func (w *WriterThatDoesNotImplementSync) Write(p []byte) (n int, err error) {
    return w.buf.Write(p)
}

func (w *WriterThatDoesNotImplementSync) Seek(offset int64, whence int) (int64, error) {
    return 0, nil
}

func TestNoPanicWhenWriterDoesNotImplementSync(t *testing.T) {
    // Given
    writer := &WriterThatDoesNotImplementSync{buf: &bytes.Buffer{}}
    var arbitraryLength int64 = 6

    // Then
    defer func() {
        if r := recover(); r != nil {
            t.Errorf("The code panicked and shouldn't have")
        }
    }()

    // When
    OverwriteStreamWithRandomBytes(writer, arbitraryLength)
}


// Seek error testing
type WriterThatDoesNotImplementSeek struct {
    buf *bytes.Buffer
}

func (w *WriterThatDoesNotImplementSeek) Write(p []byte) (n int, err error) {
    return w.buf.Write(p)
}

func TestPanicWhenWriterDoesNotImplementSeek(t *testing.T) {
    // Given
    writer := &WriterThatDoesNotImplementSeek{buf: &bytes.Buffer{}}
    var arbitraryLength int64 = 6

    // Then
    defer func() {
        if r := recover(); r == nil {
            t.Errorf("The code did not panic")
        }
    }()

    // When
    OverwriteStreamWithRandomBytes(writer, arbitraryLength)
}

type WriterThatErrorsOnSeek struct {
    buf *bytes.Buffer
}

func (w *WriterThatErrorsOnSeek) Write(p []byte) (n int, err error) {
    return w.buf.Write(p)
}

func (w *WriterThatErrorsOnSeek) Seek(offset int64, whence int) (int64, error) {
    return 0, errors.New("Some awful seek error")
}

func TestPanicsWhenErrorOnSeek(t *testing.T) {
    // Given
    writer := &WriterThatErrorsOnSeek{buf: &bytes.Buffer{}}
    var arbitraryLength int64 = 6

    // Then
    defer func() {
        if r := recover(); r == nil {
            t.Errorf("The code did not panic")
        }
    }()

    // When
    OverwriteStreamWithRandomBytes(writer, arbitraryLength)
}

// Rand read error testing
type RandReaderThatErrors struct{}
func (e RandReaderThatErrors) Read(p []byte) (n int, err error) {
    return 0, errors.New("Some awful read error")
}

func TestPanicWhenRandReadFails(t *testing.T) {
    // Given
    var arbitraryLength int64 = 6

    // Save the original rand.Reader
    originalRandReader := rand.Reader

    // Replace the rand.Read with a function that always fails
    rand.Reader = RandReaderThatErrors{}

    // Then
    defer func() {
        if r := recover(); r == nil {
            t.Errorf("The code did not panic")
        }
        rand.Reader = originalRandReader
    }()

    // When
    _ = GenerateRandomBytes(arbitraryLength)
}

// This seekable writer always resets the buffer when seeking, 
// so subsequent writes take place from the start
type SeekableWriter struct {
    buf *bytes.Buffer
}

func (b *SeekableWriter) Write(p []byte) (n int, err error) {
    return b.buf.Write(p)
}

func (b *SeekableWriter) Seek(offset int64, whence int) (int64, error) {
    b.buf.Reset()
    return 0, nil
}

func TestOverwriteStreamWithRandomBytes(t *testing.T) {
    // Given
    // Create a buffer of bytes which we're going to pass as a stream
    originalBuffer := []byte("Some bytes that need replacing")
    buffer := make([]byte, len(originalBuffer))
    copy(buffer, originalBuffer)

    // Make a seekable writer to use
    seekableWriter := &SeekableWriter{buf: bytes.NewBuffer(buffer)}
    var bufferLength int64 = int64(len(buffer))

    // When
    seekableWriter.Seek(0, 0)
    OverwriteStreamWithRandomBytes(seekableWriter, bufferLength)

    // Make sure the length is the same
    actualBufferLength := int64(len(buffer))
    if bufferLength != actualBufferLength {
        t.Errorf("Test failed, expected: '%d', got:  '%d'",
            bufferLength, actualBufferLength)
    }

    // Then
    if reflect.DeepEqual(originalBuffer, buffer) {
        t.Errorf("Test failed, expected buffers to differ")
    }
}

type WriterThatRecordsBytesWritten struct {
    buf *bytes.Buffer
    bytesWritten [][]byte
}

func (w *WriterThatRecordsBytesWritten) Write(p []byte) (n int, err error) {
    w.bytesWritten = append(w.bytesWritten, p)
    return w.buf.Write(p)
}

func (w *WriterThatRecordsBytesWritten) Seek(offset int64, whence int) (int64, error) {
    return 0, nil
}

func TestOverwriteStreamThreeTimes(t *testing.T) {
    // Given
    // Create a buffer of bytes which we're going to pass as a stream
    buffer := []byte("Some bytes that need replacing")
    bufferPtr := bytes.NewBuffer(buffer)

    bufferPtr.Reset()
    writer := &WriterThatRecordsBytesWritten{buf: bufferPtr, bytesWritten: [][]byte{}}

    // When
    OverwriteStreamWithRandomBytes(writer, int64(len(buffer)))

    // Then
    // Make sure it was overwritten the correct number of times
    if len(writer.bytesWritten) != ShredOverwriteCount {
        t.Errorf("Test failed, expected: '%d', got:  '%d'", 3, len(writer.bytesWritten))
    }
}

func TestPanicWhenFileDoesNotExist(t *testing.T) {
    // Given
    // Then
    defer func() {
        if r := recover(); r == nil {
            t.Errorf("The code did not panic")
        }
    }()

    // When
    Shred("nonexistent_file.txt")
}

func TestShredOverwritesFileWithDifferentBytesOfSameLength(t *testing.T) {
    // Temporarily switch out the file system with a memory mapped filesystem
    AppFs = afero.NewMemMapFs()

    // Given
    // Create a file (in memory - not on the filesystem)  with some content
    testString := "Some bytes that need replacing"
    file, _ := AppFs.Create("test.txt")
    file.Write([]byte(testString))
    file.Close()

    // When
    Shred("test.txt")

    // Then
    // Make sure the content is the same length after shredding
    file, _ = AppFs.Open("test.txt")
    fileInfo, _ := file.Stat()
    buffer := make([]byte, fileInfo.Size())
    file.Read(buffer)
    file.Close()

    if len(buffer) != len(testString) {
        t.Errorf("Test failed, expected: '%d', got:  '%d'", len(testString), len(buffer))
    }

    // Also make sure the buffer is different now
    if bytes.Compare(buffer, []byte(testString)) == 0 {
        t.Errorf("Test failed, expected: '%x', got:  '%x'", testString, buffer)
    }

    // Switch back the filesystem
    AppFs = afero.NewOsFs()
}

type aferoFileThatErrorsOnStat struct {
    afero.File
}

func (f *aferoFileThatErrorsOnStat) Stat() (os.FileInfo, error) {
    return nil, errors.New("Some awful stat error")
}

func TestGetFileLengthPanicsOnFailure(t *testing.T) {
    // Given
    aferoFile := &aferoFileThatErrorsOnStat{}

    // Then
    defer func() {
        if r := recover(); r == nil {
            t.Errorf("The code did not panic")
        }
    }()

    // When
    GetFileLength(aferoFile)
}

func TestGetFileLengthWithValidFile(t *testing.T) {
    AppFs = afero.NewMemMapFs()

    // Given
    // Create a file with some content
    testString := "Some bytes that need replacing"
    file, _ := AppFs.Create("test.txt")
    file.Write([]byte(testString))
    file.Close()

    // When
    file, _ = AppFs.Open("test.txt")
    length := GetFileLength(file)

    // Then
    if length != int64(len(testString)) {
        t.Errorf("Test failed, expected: '%d', got:  '%d'", len(testString), length)
    }

    AppFs = afero.NewOsFs()
}

