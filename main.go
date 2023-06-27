package main

import (
	"archive/tar"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/cosnicolaou/pbzip2"
)
const(
	dirName = "sample"
	archiveName = "samples.tar.bz2"

)

var (
	filesMap = map[int]int{
		1: 10,
		5: 5,
		30: 2,
	}
)

func main() {
	// debug
	go func() {
		fmt.Println(http.ListenAndServe(":6060", nil))
	}()

	genSampleArchieve()

	unpackTimes := 0
	http.HandleFunc("/unpack", func(w http.ResponseWriter, req *http.Request){
		runtime.GC()

		unpackTimes += 1

		os.Mkdir(dirName, 0777)
		unBzip2(archiveName, "./")
		os.RemoveAll(dirName)

		fmt.Fprintf(w, "done count: %d\n", unpackTimes)
		runtime.GC()
	})

	fmt.Printf("Start server\n")
	http.ListenAndServe(":8090", nil)
}

func genSampleArchieve() {
	fmt.Printf("Generate samples\n")
	filesPaths := genSamples()
	PackFiles(archiveName, filesPaths)
	os.RemoveAll(dirName)
	fmt.Printf("Archieve created\n")
}

func genSamples() []string {
	var files []string

	os.Mkdir(dirName, 0777)
	for size, count := range filesMap {
		for ;count > 0;count-- {
			file := fmt.Sprintf("%s/file_%d_%d", dirName, size, count)
			genBinFile(size, file)
			files = append(files, file)
		}
	}

	return files
}

func genBinFile(fileSizeMb int, fileName string){
	file, _ := os.Create(fileName)
	defer file.Close()

	fileSizeMb = fileSizeMb * 1024 * 1024
	data := make([]byte, fileSizeMb)
	rand.Read(data)
	file.Write(data)
	file.Sync()
}

func unBzip2(sourcefile, dest string) error {
	reader, err := os.Open(sourcefile)
	if err != nil {
		return err
	}
	defer reader.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	processPool := pbzip2.CreateConcurrencyPool(4)
	bzReader := pbzip2.NewReader(
		ctx,
		reader,
		pbzip2.DecompressionOptions(pbzip2.BZConcurrencyPool(processPool)),
	)
	var tarReader *tar.Reader = tar.NewReader(bzReader)

	filePath := ""
	var writer *os.File
	var header *tar.Header

	for {
		header, err = tarReader.Next()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return err
		}

		filePath = filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(filePath, os.FileMode(header.Mode))
		case tar.TypeReg, tar.TypeRegA:
			writer, err = os.Create(filePath)
			if err == nil {
				_, err = io.Copy(writer, tarReader)
				if err != nil {
					fmt.Printf("UNBZIP2: copy archieve's file error: %s \n", err)
					writer.Close()
					return err
				}
				writer.Close()

				err = os.Chmod(filePath, os.FileMode(header.Mode))
				if err != nil {
					return err
				}
			}
		default:
			fmt.Printf("UNBZIP2: provider: Unable to untar type: %c in file %s\n", header.Typeflag, filePath)
			err = errors.New("provider: Unable to untar type file" + filePath)
		}

		if err != nil {
			fmt.Printf("UNBZIP2: unpack file error: %s", err)
		}
	}

	return err
}

func PackFiles(archiveName string, files []string) error {
	args := append([]string{"-cjf", archiveName}, files...)

	cmd := exec.Command("gtar", args...)

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error while executing command: %v", err)
		return err
	}

	return nil
}
