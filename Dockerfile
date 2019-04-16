ARG \
  VERSION=7.3 \
  OS=debian \
  NAME=test
  
FROM php:$VERSION-fpm-$OS

RUN touch $NAME
