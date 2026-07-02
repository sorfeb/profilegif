# --- build stage ---
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
# Static binary, no CGO — runs anywhere.
RUN CGO_ENABLED=0 go build -o /out/profilegif .

# --- run stage ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates   # needed for HTTPS calls to the GitHub API
COPY --from=build /out/profilegif /profilegif
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/profilegif"]
