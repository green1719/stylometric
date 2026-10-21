package handler

import "stylometry/stylometry_internal/stylometry_app/stylometry_serializer"

// The types in this file describe existing JSON contracts for Swagger generation.
// Handlers continue to build their responses as before.

type SwaggerErrorResponse struct {
	Error string `json:"error"`
}

type SwaggerSampleListResponse struct {
	Items []serializer.SampleResponseDTO `json:"items"`
}

type SwaggerDraftResponse struct {
	HasDraft bool                          `json:"has_draft"`
	Draft    *serializer.SampleResponseDTO `json:"draft"`
}

type SwaggerCreatedDraftResponse struct {
	Draft serializer.SampleResponseDTO `json:"draft"`
}

type SwaggerPublishedResponse struct {
	Published bool                          `json:"published"`
	Sample    *serializer.SampleResponseDTO `json:"sample,omitempty"`
}

type SwaggerDeletedResponse struct {
	Deleted bool `json:"deleted"`
}

type SwaggerLikeRequest struct {
	StylometryLike *int `json:"stylometry_like" enums:"0,1"`
	Like           *int `json:"like" enums:"0,1"`
}

type SwaggerLikeResponse struct {
	Liked  bool                         `json:"liked"`
	Sample serializer.SampleResponseDTO `json:"sample"`
}

type SwaggerRegisterRequest struct {
	Name     string `json:"stylometry_sample_name"`
	Password string `json:"stylometry_sample_password"`
}

type SwaggerRegisteredResponse struct {
	User serializer.UserResponseDTO `json:"user"`
}

type SwaggerLoginRequest struct {
	Name     string `json:"stylometry_sample_name"`
	Password string `json:"stylometry_sample_password"`
}

type SwaggerLoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int    `json:"expires_in" example:"3600"`
}

type SwaggerLogoutResponse struct {
	Deauthenticated bool   `json:"deauthenticated"`
	AuthComment     string `json:"stylometry_auth_comment"`
}
