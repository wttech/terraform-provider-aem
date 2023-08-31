package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/wttech/terraform-provider-aem/internal/client"
)

type ClientCreateContext[T interface{}] struct {
	cl   *client.Client
	ctx  context.Context
	data T
	req  resource.CreateRequest
	resp *resource.CreateResponse
}

type ClientDeleteContext[T interface{}] struct {
	cl   *client.Client
	ctx  context.Context
	data T
	req  resource.DeleteRequest
	resp *resource.DeleteResponse
}

type ClientReadContext[T interface{}] struct {
	cl   *client.Client
	ctx  context.Context
	data T
	req  resource.ReadRequest
	resp *resource.ReadResponse
}
