package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
	"os"
)

func newSrv(t *testing.T) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(SearchServer))
	t.Cleanup(s.Close)
	return s
}

func find(t *testing.T, srv *httptest.Server, req SearchRequest) (*SearchResponse, error) {
	t.Helper()
	c := &SearchClient{AccessToken: "t", URL: srv.URL}
	return c.FindUsers(req)
}

func TestFindUsersOK(t *testing.T) {
	srv := newSrv(t)

	r, err := find(t, srv, SearchRequest{Limit: 2, OrderField: "Id", OrderBy: OrderByAsc})
	if err != nil || !r.NextPage || len(r.Users) != 2 {
		t.Fatalf("NextPage case: %+v err=%v", r, err)
	}

	r, err = find(t, srv, SearchRequest{
    Limit:      100,
    Offset:     30,
    OrderField: "Id",
    OrderBy:    OrderByAsc,
	})
	if err != nil || r.NextPage {
		t.Fatalf("last page case: %+v err=%v", r, err)
	}
	if len(r.Users) != 5 {
		t.Fatalf("expected 5 users, got %d", len(r.Users))
	}

	r2, err := find(t, srv, SearchRequest{Limit: 2,
										  Offset: 2,
										  OrderField: "Id",
										  OrderBy: OrderByAsc})
	if err != nil || r2.Users[0].ID == r.Users[0].ID{
		t.Fatalf("offset didn't work: r=%+v r2=%+v err=%v", r, r2, err)
	}
}

func TestFindUsersClientErrors(t *testing.T){
	if _, err := (&SearchClient{}).FindUsers(SearchRequest{Limit: -1}); err == nil ||
		!strings.Contains(err.Error(), "limit"){
			t.Errorf("limit<0: %v", err)
	}

	if _, err := (&SearchClient{}).FindUsers(SearchRequest{Offset: -1}); err == nil ||
		!strings.Contains(err.Error(), "offset"){
		t.Errorf("offset<0: %v", err)
	}

	cases := []struct {
	name    string
	handler http.HandlerFunc
	wantSub string
}{
		{"401", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
		}, "bad AccessToken"},
		{"500", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}, "fatal error"},
		{"400 other", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(SearchErrorResponse{Error: "oops"})
		}, "unknown bad request"},
		{"400 bad json", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(400)
			w.Write([]byte("not-json"))
		}, "cant unpack error json"},
		{"200 bad json", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not-json"))
		}, "cant unpack result json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()
			_, err := (&SearchClient{URL: srv.URL}).FindUsers(SearchRequest{Limit: 1, OrderField: "Id"})
			if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("want %q, got %v", tc.wantSub, err)
			}
		})
	}
}

func TestFindUsersTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()
	_, err := (&SearchClient{URL: srv.URL}).FindUsers(SearchRequest{Limit: 1, OrderField: "Id"})
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("want timeout, got %v", err)
	}
}

func TestSearchServerSortsAndFilters(t *testing.T) {
	srv := newSrv(t)

	checkSorted := func(t *testing.T, users []User, field string, by int) {
		t.Helper()
		for i := 1; i < len(users); i++ {
			var a, b int
			switch field {
			case "Id":
				a, b = users[i-1].ID, users[i].ID
			case "Age":
				a, b = users[i-1].Age, users[i].Age
			case "Name", "":
				if (by == OrderByAsc && users[i-1].Name > users[i].Name) ||
					(by == OrderByDesc && users[i-1].Name < users[i].Name) {
					t.Fatalf("not sorted %s by %s: %v", field, users[i-1].Name, users[i].Name)
				}
				continue
			}
			if by == OrderByAsc && a > b {
				t.Fatalf("not asc %s: %d > %d", field, a, b)
			}
			if by == OrderByDesc && a < b {
				t.Fatalf("not desc %s: %d < %d", field, a, b)
			}
		}
	}

	for _, tc := range []struct {
		field string
		by    int
	}{
		{"Id", OrderByAsc}, {"Id", OrderByDesc},
		{"Age", OrderByAsc}, {"Age", OrderByDesc},
		{"Name", OrderByAsc}, {"Name", OrderByDesc},
		{"", OrderByAsc}, 
	} {
		t.Run(tc.field+"/"+strconv.Itoa(tc.by), func(t *testing.T) {
			r, err := find(t, srv, SearchRequest{Limit: 10, OrderField: tc.field, OrderBy: tc.by})
			if err != nil {
				t.Fatal(err)
			}
			checkSorted(t, r.Users, tc.field, tc.by)
		})
	}

	r, err := find(t, srv, SearchRequest{Limit: 5, OrderField: "Name", OrderBy: OrderByAsIs})
	if err != nil || len(r.Users) == 0 {
		t.Fatalf("as is: %+v %v", r, err)
	}

	for _, q := range []string{"Boyd", "labore"} {
		r, err := find(t, srv, SearchRequest{Limit: 50, Query: q, OrderField: "Name"})
		if err != nil || len(r.Users) == 0 {
			t.Fatalf("query %q: %+v %v", q, r, err)
		}
		for _, u := range r.Users {
			lu := strings.ToLower(u.Name + " " + u.About)
			if !strings.Contains(lu, strings.ToLower(q)) {
				t.Fatalf("user %v doesn't match %q", u, q)
			}
		}
	}

	r, _ = find(t, srv, SearchRequest{Limit: 1, Query: "Boyd", OrderField: "Name"})
	if r.Users[0].Name != "Boyd Wolf" || r.Users[0].Age != 22 {
		t.Fatalf("real data mismatch: %+v", r.Users[0])
	}
}


func TestSearchServerErrors(t *testing.T) {
	srv := newSrv(t)

	// bad order_field -> 400
	_, err := find(t, srv, SearchRequest{Limit: 1, OrderField: "Nope", OrderBy: OrderByAsc})
	if err == nil || !strings.Contains(err.Error(), "OrderFeld") {
		t.Fatalf("bad order_field: %v", err)
	}

	resp, err := http.Get(srv.URL + "?limit=abc&offset=xyz&order_field=Name&order_by=1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var users []User
	json.NewDecoder(resp.Body).Decode(&users)
	if len(users) != 0 {
		t.Fatalf("expected empty, got %d", len(users))
	}

	old := DatasetFile
	DatasetFile = "nonexistent.xml"
	defer func() { DatasetFile = old }()
	_, err = find(t, srv, SearchRequest{Limit: 1, OrderField: "Id"})
	if err == nil || !strings.Contains(err.Error(), "fatal error") {
		t.Fatalf("missing file: %v", err)
	}
}

func TestReadUsersAndWriteError(t *testing.T) {
	users, err := readUsers("dataset.xml")
	if err != nil || len(users) != 35 {
		t.Fatalf("readUsers: %d %v", len(users), err)
	}
	if _, err := readUsers("nope.xml"); err == nil {
		t.Fatal("expected error")
	}

	w := httptest.NewRecorder()
	writeError(w, 400, "x")
	if w.Code != 400 || !strings.Contains(w.Body.String(), "x") {
		t.Fatalf("writeError: %d %s", w.Code, w.Body.String())
	}
}

func TestReadUsersMalformedXML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "bad_*.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(`<root><row><id>not_an_int</id></row></root>`)
	tmpFile.Close()

	_, err = readUsers(tmpFile.Name())
	if err == nil {
		t.Fatal("expected error for malformed xml")
	}
}