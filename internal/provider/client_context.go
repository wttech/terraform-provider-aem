package provider

import (
	"context"
	"github.com/wttech/terraform-provider-aem/internal/client"
)

type ClientContext[T interface{}] struct {
	cl   *client.Client
	ctx  context.Context
	data T
}
