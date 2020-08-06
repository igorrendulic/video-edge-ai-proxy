### STAGE 1: Build ###
FROM node:12.7-alpine AS build
WORKDIR /usr/src/app
COPY package.json package-lock.json ./
RUN npm install
COPY . .
RUN npm run build

### STAGE 2: Run ###
FROM nginx:1.17.1-alpine

LABEL maintainer Igor Rendulic <igor@chryscloud.com>

COPY nginx.conf /etc/nginx/nginx.conf
COPY --from=build /usr/src/app/dist/web-edgeproxy-portal /usr/share/nginx/html