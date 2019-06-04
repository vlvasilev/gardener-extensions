#############      builder-base                             #############
FROM golang:1.11.5 AS builder-base

COPY ./hack/install-requirements.sh /install-requirements.sh

RUN /install-requirements.sh

#############      builder                                  #############
FROM builder-base AS builder

ARG VERIFY=false

WORKDIR /go/src/github.com/gardener/gardener-extensions
COPY . .

RUN make all

#############      base                                     #############
FROM alpine:3.8 AS base

RUN apk add --update bash curl

WORKDIR /

#############      gardener-extension-hyper                 #############
FROM base AS gardener-extension-hyper

COPY controllers/provider-aws/charts /controllers/provider-aws/charts
COPY controllers/provider-azure/charts /controllers/provider-azure/charts
COPY controllers/provider-gcp/charts /controllers/provider-gcp/charts
COPY controllers/provider-openstack/charts /controllers/provider-openstack/charts
COPY controllers/provider-alicloud/charts /controllers/provider-alicloud/charts
COPY controllers/provider-local/charts /controllers/provider-local/charts

COPY --from=builder /go/bin/gardener-extension-hyper /gardener-extension-hyper

ENTRYPOINT ["/gardener-extension-hyper"]
