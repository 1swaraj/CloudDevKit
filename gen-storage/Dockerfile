FROM gcr.io/google.com/cloudsdktool/cloud-sdk:alpine as build
# install build tools
RUN apk add go git
# cache dependencies
ADD ../go.mod go.sum ./
RUN go mod download GOPROXY=direct
# build
ADD ../../../go-cloud%20copy/gocdk-blob .
RUN go build -o /main
# copy artifacts to a clean image
FROM alpine
COPY --from=build /main /main
ENTRYPOINT [ "/main" ]