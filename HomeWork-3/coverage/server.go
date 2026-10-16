package main

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)
var DatasetFile = "dataset.xml"


type Dataset struct {
	XMLName xml.Name  `xml:"root"`
	Users   []UserXML `xml:"row"`
}

type UserXML struct{
	ID     int	`xml:"id"`
	FirstName   string `xml:"first_name"`
	LastName	string `xml:"last_name"`
	Age    int	`xml:"age"`
	About  string	`xml:"about"`
	Gender string	`xml:"gender"`
}

func writeError(w http.ResponseWriter, status int, msg string){
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := SearchErrorResponse{Error: msg}
	b, _ := json.Marshal(resp)
	w.Write(b)
}

func readUsers(path string) ([]UserXML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var dataset Dataset
	if err := xml.Unmarshal(data, &dataset); err != nil {
		return nil, err
	}

	return dataset.Users, nil
} 

func SearchServer(w http.ResponseWriter, r *http.Request) {
	// Тут писать SearchServer
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 0{
		limit = 0
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || limit < 0{
		offset = 0
	}

	query := r.URL.Query().Get("query")
	orderField := r.URL.Query().Get("order_field")
	orderBy, err := strconv.Atoi(r.URL.Query().Get("order_by"))
	if err != nil{
		orderBy = OrderByAsIs
	}

	valid := map[string]bool{
		"":		true,
		"Id":	true,
		"Age":	true,
		"Name":	true,
	}

	if !valid[orderField]{
		writeError(w, http.StatusBadRequest, ErrorBadOrderField)
		return
	}


	if orderField == "" {
		orderField = "Name"
	}

	users, err := readUsers(DatasetFile)
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userRes := make([]User, 0, len(users))
	for _, u := range users{
		userRes = append(userRes, User{
			ID: u.ID,
			Name: u.FirstName + " " + u.LastName,
			Age: u.Age,
			About: u.About,
			Gender: u.Gender,
		})
	}

	if query != ""{
		filtered := make([]User, 0)
		lowerQuery := strings.ToLower(query)
		for _, u := range userRes{
			if strings.Contains(strings.ToLower(u.Name), lowerQuery) ||
					strings.Contains(strings.ToLower(u.About), lowerQuery){
						filtered = append(filtered, u)
			}
		}

		userRes = filtered
	}

	if orderBy != OrderByAsIs{
		sort.SliceStable(userRes, func(i, j int) bool {
			var less bool
			switch orderField {
			case "Id":
				less = userRes[i].ID < userRes[j].ID
			case "Age":
				less = userRes[i].Age < userRes[j].Age
			case "Name":
				less = userRes[i].Name < userRes[j].Name
			}

			if orderBy == OrderByDesc{
				return !less
			}

			return less
		})
	}

	if offset > len(userRes) {
		offset = len(userRes)
	}

	end := offset + limit
	if end > len(userRes) {
		end = len(userRes)
	}
	userRes = userRes[offset:end]

	
	b, err := json.Marshal(userRes)
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(b); err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
