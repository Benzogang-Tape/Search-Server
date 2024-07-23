package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/gorilla/schema"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"
)

const (
	userTagName   = "row"
	ageFieldName  = "age"
	nameFieldName = "name"
	idFieldName   = "id"
)

var (
	database             = "dataset.xml"
	badOrderFieldMessage = "{" + "\"error\": " + "\"" + ErrorBadOrderField + "\"" + "}"
	pauseDuration        = time.Second * 2
	errParsingXMLFailed  = errors.New("failed to parse file")
	errBadOffsetParam    = errors.New("{\"error\": \"bad offset param\"}")
	errBadQueryParams    = errors.New("{\"error\": \"bad query params\"}")
)

func SearchServer(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("AccessToken")
	if strings.Compare(token, "") == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	users := make([]User, 0)
	prms := r.URL.Query()
	params := new(SearchRequest)
	decoder := schema.NewDecoder()
	if err := decoder.Decode(params, prms); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, errBadQueryParams.Error())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	if params.OrderField != "" && params.OrderField != nameFieldName && params.OrderField != ageFieldName && params.OrderField != idFieldName {
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, badOrderFieldMessage)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	data, err := os.ReadFile(database)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = parseUsers(data, &users)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error()) //nolint:errcheck
		return
	}
	users, err = sortUsers(users, *params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = io.WriteString(w, err.Error())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(users); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func parseUsers(data []byte, users *[]User) error {
	user := User{}
	xmlData := strings.NewReader(string(data))
	d := xml.NewDecoder(xmlData)
	for t, err := d.Token(); t != nil; t, err = d.Token() {
		if err != nil {
			return errParsingXMLFailed
		}
		if tokenType, ok := t.(xml.StartElement); ok && tokenType.Name.Local == userTagName {
			err := parseUser(d, &user)
			if err != nil {
				return err
			}
			*users = append(*users, user)
		}
	}
	return nil
}

func parseUser(d *xml.Decoder, user *User) error {
	for tag, err := d.Token(); tag != nil; tag, err = d.Token() {
		if err != nil {
			return errParsingXMLFailed
		}
		if tagType, ok := tag.(xml.StartElement); ok {
			switch tagType.Name.Local {
			case "id":
				if err := d.DecodeElement(&user.ID, &tagType); err != nil {
					return errParsingXMLFailed
				}
			case "age":
				if err := d.DecodeElement(&user.Age, &tagType); err != nil {
					return errParsingXMLFailed
				}
			case "first_name":
				if err := d.DecodeElement(&user.Name, &tagType); err != nil {
					return errParsingXMLFailed
				}
			case "last_name":
				surname := ""
				if err := d.DecodeElement(&surname, &tagType); err != nil {
					return errParsingXMLFailed
				}
				user.Name += " " + surname
			case "about":
				if err := d.DecodeElement(&user.About, &tagType); err != nil {
					return errParsingXMLFailed
				}
				user.About = user.About[:len(user.About)-1]
			case "gender":
				if err := d.DecodeElement(&user.Gender, &tagType); err != nil {
					return errParsingXMLFailed
				}
			case "favoriteFruit":
				return nil
			}
		}
	}
	return nil
}

func sortUsers(users []User, params SearchRequest) ([]User, error) {
	compareUsersByAge := func(a, b User) int {
		if a.Age < b.Age {
			return -1
		}
		if a.Age > b.Age {
			return 1
		}
		return 0
	}
	compareUsersByID := func(a, b User) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	}
	sortByFieldAndOrder := func(users []User, order int, sortFunc func(a, b User) int) {
		if order < 0 {
			slices.SortStableFunc(users, func(aa, bb User) int {
				return -sortFunc(aa, bb)
			})
		} else {
			slices.SortStableFunc(users, sortFunc)
		}
	}
	if params.Query != "" {
		users = slices.DeleteFunc(users, func(item User) bool {
			return !strings.Contains(item.Name, params.Query) && !strings.Contains(item.About, params.Query)
		})
	}
	if params.Offset >= len(users) {
		return nil, errBadOffsetParam
	}
	if params.OrderBy != 0 {
		switch params.OrderField {
		case "":
			fallthrough
		case nameFieldName:
			sortByFieldAndOrder(users, params.OrderBy, func(a, b User) int {
				return strings.Compare(a.Name, b.Name)
			})
		case ageFieldName:
			sortByFieldAndOrder(users, params.OrderBy, compareUsersByAge)
		case idFieldName:
			sortByFieldAndOrder(users, params.OrderBy, compareUsersByID)
		}
	}
	if lastUserIdx := params.Offset + params.Limit; lastUserIdx <= len(users) {
		users = users[params.Offset:lastUserIdx]
	} else {
		users = users[params.Offset:]
	}
	return users, nil
}
