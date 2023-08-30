package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/wttech/terraform-provider-aem/internal/client"
)

type ClientContext[T interface{}] struct {
	data T
	ctx  context.Context
	req  resource.CreateRequest
	resp *resource.CreateResponse
	cl   client.Client
}
