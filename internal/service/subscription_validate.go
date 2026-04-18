package service

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/wassiliy/subscriptions-service/internal/apperrors"
	"github.com/wassiliy/subscriptions-service/internal/domain"
	"github.com/wassiliy/subscriptions-service/internal/pkg/month"
)

var createSubscriptionValidator *validator.Validate

func init() {
	v := validator.New()
	_ = v.RegisterValidation("month", validateMonthTag)
	_ = v.RegisterValidation("go_uuid", validateGoUUIDTag)
	createSubscriptionValidator = v
}

func validateMonthTag(fl validator.FieldLevel) bool {
	field := fl.Field()
	switch field.Kind() {
	case reflect.String:
		s := field.String()
		if s == "" {
			return true
		}
		_, err := month.Parse(s)
		return err == nil
	case reflect.Pointer:
		if field.IsNil() {
			return true
		}
		elem := field.Elem()
		if elem.Kind() != reflect.String {
			return false
		}
		s := elem.String()
		if s == "" {
			return true
		}
		_, err := month.Parse(s)
		return err == nil
	default:
		return false
	}
}

func validateGoUUIDTag(fl validator.FieldLevel) bool {
	u, ok := fl.Field().Interface().(uuid.UUID)
	if !ok {
		return false
	}
	return u != uuid.Nil
}

func validateCreatePayload(in domain.CreateSubscription) error {
	err := createSubscriptionValidator.Struct(in)
	if err == nil {
		return nil
	}
	return wrapValidationErrors(err)
}

func wrapValidationErrors(err error) error {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		var parts []string
		for _, e := range verrs {
			parts = append(parts, formatValidationError(e))
		}
		return fmt.Errorf("%w: %s", apperrors.ErrInvalidArgument, strings.Join(parts, "; "))
	}
	return err
}

func formatValidationError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", e.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s", e.Field(), e.Param())
	case "month":
		return fmt.Sprintf("%s must be MM-YYYY", e.Field())
	case "go_uuid":
		return fmt.Sprintf("%s must be a valid non-zero UUID", e.Field())
	default:
		return e.Error()
	}
}
