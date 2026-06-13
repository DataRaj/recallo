package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"recallo/internals/logger"
	"recallo/internals/middleware"
	"recallo/internals/utils"
)

func HandleFileUpload(w http.ResponseWriter, r *http.Request) {
	senderID, ok := r.Context().Value(middleware.CtxUserID).(int64)
	if !ok {
		logger.App.Printf("[FILE_UPLOAD] error=missing_user_id_in_context remote=%s", r.RemoteAddr)
		utils.JSON(w, http.StatusUnauthorized, false, "Unauthorized user", nil)
		return
	}

	fetchedID := r.PathValue("private_id")
	privateID, err := strconv.ParseInt(fetchedID, 10, 64)
	if err != nil {
		logger.App.Printf("[FILE_UPLOAD] error=invalid_private_id raw=%q sender_id=%d", fetchedID, senderID)
		utils.JSON(w, http.StatusBadRequest, false, "invalid private id", nil)
		return
	}

	logger.App.Printf("[FILE_UPLOAD] upload_started sender_id=%d private_id=%d remote=%s", senderID, privateID, r.RemoteAddr)

	err = r.ParseMultipartForm(50 << 20)
	if err != nil {
		logger.App.Printf("[FILE_UPLOAD] error=parse_multipart_form sender_id=%d private_id=%d err=%v", senderID, privateID, err)
		utils.JSON(w, http.StatusBadRequest, false, "failed to Parse Multipart Form", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		logger.App.Printf("[FILE_UPLOAD] error=retrieve_form_file sender_id=%d private_id=%d err=%v", senderID, privateID, err)
		utils.JSON(w, http.StatusBadRequest, false, "Failed to retrieve file from form", nil)
		return
	}
	defer file.Close()

	dirPath := filepath.Join("files", "chats", fmt.Sprintf("%d", privateID), fmt.Sprintf("%d", senderID))
	logger.App.Printf("[FILE_UPLOAD] dir_path=%s filename=%s sender_id=%d private_id=%d", dirPath, header.Filename, senderID, privateID)

	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		logger.App.Printf("[FILE_UPLOAD] error=mkdir_all dir=%s sender_id=%d private_id=%d err=%v", dirPath, senderID, privateID, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to create directory", nil)
		return
	}

	filePath := filepath.Join(dirPath, header.Filename)
	outFile, err := os.Create(filePath)
	if err != nil {
		logger.App.Printf("[FILE_UPLOAD] error=create_file path=%s sender_id=%d private_id=%d err=%v", filePath, senderID, privateID, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to create a file", nil)
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		logger.App.Printf("[FILE_UPLOAD] error=copy_file path=%s sender_id=%d private_id=%d err=%v", filePath, senderID, privateID, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to save file", nil)
		return
	}

	fileUrl := fmt.Sprintf("/files/chats/%d/%d/%s", privateID, senderID, header.Filename)
	logger.App.Printf("[FILE_UPLOAD] upload_complete sender_id=%d private_id=%d file_url=%s", senderID, privateID, fileUrl)

	utils.JSON(w, http.StatusOK, true, "file uploaded successfully", fileUrl)
}

func HandleGetFile() http.Handler {
	logger.App.Printf("[FILE_SERVER] static file server mounted at ./files, strip_prefix=api/v1/files")
	fs := http.FileServer(http.Dir("./files"))
	return http.StripPrefix("/api/v1/files", fs)
}
