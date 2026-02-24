package feed

import "fmt"

// FeedID is a value object representing a feed/note identifier.
type FeedID struct {
	value string
}

// NewFeedID creates a FeedID, returning an error if id is empty.
func NewFeedID(id string) (FeedID, error) {
	if id == "" {
		return FeedID{}, fmt.Errorf("feed ID 不能为空")
	}
	return FeedID{value: id}, nil
}

// String returns the underlying feed ID string.
func (f FeedID) String() string {
	return f.value
}

// XsecToken is a value object representing a security token for feed access.
type XsecToken struct {
	value string
}

// NewXsecToken creates an XsecToken, returning an error if token is empty.
func NewXsecToken(token string) (XsecToken, error) {
	if token == "" {
		return XsecToken{}, fmt.Errorf("xsec_token 不能为空")
	}
	return XsecToken{value: token}, nil
}

// String returns the underlying token string.
func (t XsecToken) String() string {
	return t.value
}
