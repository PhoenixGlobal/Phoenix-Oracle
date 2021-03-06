package web

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/manyminds/api2go/jsonapi"
	"github.com/pkg/errors"
)

const (
	PaginationDefault = 25

	MediaType = "application/vnd.api+json"

	KeyNextLink = "next"

	KeyPreviousLink = "prev"
)

func ParsePaginatedRequest(sizeParam, pageParam string) (int, int, int, error) {
	var err error
	page := 1
	size := PaginationDefault

	if sizeParam != "" {
		if size, err = strconv.Atoi(sizeParam); err != nil || size < 1 {
			return 0, 0, 0, fmt.Errorf("invalid size param, error: %+v", err)
		}
	}

	if pageParam != "" {
		if page, err = strconv.Atoi(pageParam); err != nil || page < 1 {
			return 0, 0, 0, fmt.Errorf("invalid page param, error: %+v", err)
		}
	}

	offset := (page - 1) * size
	return size, page, offset, nil
}

func paginationLink(url url.URL, size, page int) jsonapi.Link {
	query := url.Query()
	query.Set("size", strconv.Itoa(size))
	query.Set("page", strconv.Itoa(page))
	url.RawQuery = query.Encode()
	return jsonapi.Link{Href: url.String()}
}

func nextLink(url url.URL, size, page int) jsonapi.Link {
	return paginationLink(url, size, page+1)
}

func prevLink(url url.URL, size, page int) jsonapi.Link {
	return paginationLink(url, size, page-1)
}

func NewJSONAPIResponse(resource interface{}) ([]byte, error) {
	document, err := jsonapi.MarshalToStruct(resource, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource to struct: %+v", err)
	}

	return json.Marshal(document)
}

func NewPaginatedResponseWithMeta(url url.URL, size, page, count int, resource interface{}, meta map[string]interface{}) ([]byte, error) {
	document, err := getPaginatedResponseDoc(url, size, page, count, resource)
	if err != nil {
		return nil, err
	}
	if document.Meta == nil {
		document.Meta = make(jsonapi.Meta)
	}
	for key, val := range meta {
		document.Meta[key] = val
	}
	return json.Marshal(document)
}

func NewPaginatedResponse(url url.URL, size, page, count int, resource interface{}) ([]byte, error) {
	document, err := getPaginatedResponseDoc(url, size, page, count, resource)
	if err != nil {
		return nil, err
	}
	return json.Marshal(document)
}

func getPaginatedResponseDoc(url url.URL, size, page, count int, resource interface{}) (*jsonapi.Document, error) {
	document, err := jsonapi.MarshalToStruct(resource, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource to struct: %+v", err)
	}

	document.Meta = make(jsonapi.Meta)
	document.Meta["count"] = count

	document.Links = make(jsonapi.Links)
	if count > size {
		if page*size < count {
			document.Links[KeyNextLink] = nextLink(url, size, page)
		}
		if page > 1 {
			document.Links[KeyPreviousLink] = prevLink(url, size, page)
		}
	}
	return document, nil
}

func ParsePaginatedResponse(input []byte, resource interface{}, links *jsonapi.Links) error {
	document := jsonapi.Document{}
	err := parsePaginatedResponseToDocument(input, resource, &document)
	if err != nil {
		return err
	}
	*links = document.Links
	return nil
}

func ParsePaginatedResponseWithMeta(input []byte, resource interface{}, links *jsonapi.Links, meta *jsonapi.Meta) error {
	document := jsonapi.Document{}
	err := parsePaginatedResponseToDocument(input, resource, &document)
	if err != nil {
		return err
	}
	*links = document.Links
	*meta = document.Meta
	return nil
}

func parsePaginatedResponseToDocument(input []byte, resource interface{}, document *jsonapi.Document) error {
	err := ParseJSONAPIResponse(input, resource)
	if err != nil {
		return errors.Wrap(err, "ParseJSONAPIResponse error")
	}

	// Unmarshal using the stdlib Unmarshal to extract the links part of the document
	err = json.Unmarshal(input, &document)
	if err != nil {
		return fmt.Errorf("unable to unmarshal links: %+v", err)
	}
	return nil
}

func ParseJSONAPIResponse(input []byte, resource interface{}) error {
	// as is api2go will discard the links
	err := jsonapi.Unmarshal(input, resource)
	if err != nil {
		return fmt.Errorf("web: unable to unmarshal data of type %T, %+v", resource, err)
	}

	return nil
}
