package identity

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	identityv1 "staffino/gen/staffino/identity/v1"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RequestOTP(ctx context.Context, req *connect.Request[identityv1.RequestOTPRequest]) (*connect.Response[identityv1.RequestOTPResponse], error) {
	seconds, err := h.svc.RequestOTP(ctx, req.Msg.Phone)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&identityv1.RequestOTPResponse{
		ExpiresInSeconds: seconds,
	}), nil
}

func (h *Handler) VerifyOTP(ctx context.Context, req *connect.Request[identityv1.VerifyOTPRequest]) (*connect.Response[identityv1.VerifyOTPResponse], error) {
	result, err := h.svc.VerifyOTP(ctx, req.Msg.Phone, req.Msg.Otp)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&identityv1.VerifyOTPResponse{
		User:                   toProto(result.User),
		AccessToken:            result.AccessToken,
		AccessTokenExpiresAt:   timestamppb.New(result.AccessTokenExpiresAt),
	}), nil
}

func (h *Handler) GetUser(ctx context.Context, req *connect.Request[identityv1.GetUserRequest]) (*connect.Response[identityv1.GetUserResponse], error) {
	user, err := h.svc.GetUser(ctx, req.Msg.Id)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&identityv1.GetUserResponse{
		User: toProto(*user),
	}), nil
}

func toProto(user User) *identityv1.User {
	return &identityv1.User{
		Id:        user.ID,
		Phone:     user.Phone,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
}

func connectErr(err error) error {
	switch {
	case errors.Is(err, ErrValidation):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrUnauthorized):
		return connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrConflict):
		return connect.NewError(connect.CodeAlreadyExists, err)
	default:
		return connect.NewError(connect.CodeInternal, errors.New("internal"))
	}
}