package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

type SearchRequestServer struct {
	Limit      int
	Offset     int
	Query      string
	OrderField string
	OrderBy    int
}

type UsersServer struct {
	Users []UserServer `xml:"row"`
}

type UserServer struct {
	ID      int    `xml:"id"`
	Name    string `xml:"first_name"`
	Surname string `xml:"last_name"`
	Age     int    `xml:"age"`
	About   string `xml:"about"`
	Gender  string `xml:"gender"`
}

type UserClient struct {
	ID     int
	Name   string
	Age    int
	About  string
	Gender string
}

type ErrorServer struct {
	Error string `json:"error"`
}

const (
	ageFieldName  = "age"
	nameFieldName = "name"
	idFieldName   = "id"
)

var (
	SecretToken           = []byte("secret")
	database              = "dataset.xml"
	errBadOrderFieldParam = errors.New(ErrorBadOrderField)
	errParsingXMLFailed   = errors.New("failed to parse file")
	errBadLimitParam      = errors.New("bad limit param")
	errBadOffsetParam     = errors.New("bad offset param")
	errBadOrderByParam    = errors.New("bad order_by param")
	errBadQueryParams     = errors.New("bad query params")
	errBadAccessToken     = errors.New("bad AccessToken")
)

func SearchServer(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("AccessToken")
	err := authCheck(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	enc := json.NewEncoder(w)

	sendErrorResponse := func(errMsg string, statusCode int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		Msg := ErrorServer{Error: errMsg}
		if err = enc.Encode(Msg); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	rawParams := r.URL.Query()
	params, err := parseQueryParams(rawParams)
	if err != nil {
		sendErrorResponse(err.Error(), http.StatusBadRequest)
		return
	}

	err = validateQueryParams(params)
	if err != nil {
		log.Printf("validateQueryParams: %s\n", err.Error())
		sendErrorResponse(err.Error(), http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(database)
	if err != nil {
		log.Printf("SearchServer: Failed to read %s: %s\n", database, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	users, err := parseUsers(data)
	if err != nil {
		log.Printf("parseUsers: %s\n", err.Error())
		sendErrorResponse(err.Error(), http.StatusInternalServerError)
		return
	}

	users = processUsers(users, *params)

	w.Header().Set("Content-Type", "application/json")
	if err = enc.Encode(users); err != nil {
		log.Printf("SearchServer: Failed to send response: %s\n", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func authCheck(token string) error {
	hashSecretGetter := func(token *jwt.Token) (interface{}, error) {
		return SecretToken, nil
	}

	parsedToken, err := jwt.Parse(token, hashSecretGetter)
	if err != nil || !parsedToken.Valid {
		return errBadAccessToken
	}

	return nil
}

func parseQueryParams(rawParams url.Values) (*SearchRequestServer, error) {
	rawLimit := rawParams.Get("limit")
	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		return nil, errBadQueryParams
	}

	rawOffset := rawParams.Get("offset")
	offset, err := strconv.Atoi(rawOffset)
	if err != nil {
		return nil, errBadQueryParams
	}

	rawOrderBy := rawParams.Get("order_by")
	orderBy, err := strconv.Atoi(rawOrderBy)
	if err != nil {
		return nil, errBadQueryParams
	}

	return &SearchRequestServer{
		Limit:      limit,
		Offset:     offset,
		Query:      rawParams.Get("query"),
		OrderField: rawParams.Get("order_field"),
		OrderBy:    orderBy,
	}, nil
}

func validateQueryParams(params *SearchRequestServer) error {
	if params.OrderField != "" && params.OrderField != nameFieldName &&
		params.OrderField != ageFieldName && params.OrderField != idFieldName {
		return errBadOrderFieldParam
	}
	if params.Limit <= 0 {
		return errBadLimitParam
	}
	if params.Offset < 0 {
		return errBadOffsetParam
	}
	switch params.OrderBy {
	case -1, 0, 1:
		return nil
	default:
		return errBadOrderByParam
	}
}

func parseUsers(data []byte) ([]UserClient, error) {
	users := UsersServer{}
	err := xml.Unmarshal(data, &users)
	if err != nil {
		return nil, errParsingXMLFailed
	}

	unexpectedChars := string([]rune{10, 32})
	parsedUsers := make([]UserClient, 0, len(users.Users))
	for _, user := range users.Users {
		parsedUsers = append(parsedUsers, UserClient{
			ID:     user.ID,
			Name:   fmt.Sprintf("%s %s", user.Name, user.Surname),
			Age:    user.Age,
			About:  strings.TrimRight(user.About, unexpectedChars),
			Gender: user.Gender,
		})
	}
	return parsedUsers, nil
}

func processUsers(users []UserClient, params SearchRequestServer) []UserClient {
	users = filterUsers(users, params.Query)
	if params.Offset >= len(users) {
		return []UserClient{}
	}

	users = sortUsers(users, params.OrderField, params.OrderBy)
	users = paginateUsers(users, params.Offset, params.Limit)
	return users
}

func filterUsers(users []UserClient, query string) []UserClient {
	if query != "" {
		users = slices.DeleteFunc(users, func(item UserClient) bool {
			return !strings.Contains(item.Name, query) && !strings.Contains(item.About, query)
		})
	}
	return users
}

func sortUsers(users []UserClient, orderField string, orderBy int) []UserClient {
	sortByFieldAndOrder := func(users []UserClient, order int, sortFunc func(a, b UserClient) int) {
		if order < 0 {
			slices.SortStableFunc(users, func(aa, bb UserClient) int {
				return -sortFunc(aa, bb)
			})
		} else {
			slices.SortStableFunc(users, sortFunc)
		}
	}

	compareUsersByAge := func(a, b UserClient) int {
		if a.Age < b.Age {
			return -1
		}
		if a.Age > b.Age {
			return 1
		}
		return 0
	}

	compareUsersByID := func(a, b UserClient) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	}

	if orderBy != 0 {
		switch orderField {
		case "", nameFieldName:
			sortByFieldAndOrder(users, orderBy, func(a, b UserClient) int {
				return strings.Compare(a.Name, b.Name)
			})
		case ageFieldName:
			sortByFieldAndOrder(users, orderBy, compareUsersByAge)
		case idFieldName:
			sortByFieldAndOrder(users, orderBy, compareUsersByID)
		}
	}

	return users
}

func paginateUsers(users []UserClient, offset, limit int) []UserClient {
	if lastUserIdx := offset + limit; lastUserIdx <= len(users) {
		users = users[offset:lastUserIdx]
	} else {
		users = users[offset:]
	}
	return users
}
