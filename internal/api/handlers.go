package api

import (
	"net/http"
	"github.com/vasantbala/notebook-service/internal/util"
)

func ListNotebooksHandler(w http.ResponseWriter, r *http.Request){
	util.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "list notebooks",
	})
}

func CreateNotebookHandler(w http.ResponseWriter, r *http.Request){
	util.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "create notebooks",
	})
}