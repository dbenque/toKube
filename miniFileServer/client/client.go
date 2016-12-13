package client

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
)

//PostFile post a file to the miniFileServer
func PostFile(filepath string, targetURL string) error {
	pwd, _ := os.Getwd()
	defer os.Chdir(pwd)

	filedir, filename := path.Split(filepath)
	os.Chdir(filedir)

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	// this step is very important
	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", filename)
	if err != nil {
		fmt.Println("error writing to buffer")
		return err
	}

	// open file handle
	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening file")
		return err
	}

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(targetURL, contentType, bodyBuf)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Error returned by server: %#v", resp)
	}
	return nil
}
