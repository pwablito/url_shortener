FROM golang:1.16

COPY . /app

WORKDIR /app

RUN go build .

CMD /app/url_shortener
