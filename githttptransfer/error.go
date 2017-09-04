package githttptransfer

import "fmt"

type UrlNotFoundError struct {
	Method string
	Path   string
}

func (e *UrlNotFoundError) Error() string {
	return fmt.Sprintf("Url Not Found: Method %s, Path %s", e.Method, e.Path)
}

type MethodNotAllowedError struct {
	Method string
	Path   string
}

func (e *MethodNotAllowedError) Error() string {
	return fmt.Sprintf("Method Not Allowed: Method %s, Path %s", e.Method, e.Path)
}

type NoAccessError struct {
	Dir string
}

func (e *NoAccessError) Error() string {
	return "No Access: " + e.Dir
}
