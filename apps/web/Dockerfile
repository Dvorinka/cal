FROM node:22-alpine AS build
WORKDIR /src
COPY package.json package-lock.json* ./
COPY apps/web/package.json apps/web/package.json
COPY packages/api-client/package.json packages/api-client/package.json
RUN npm install
COPY packages/api-client packages/api-client
COPY apps/web apps/web
RUN npm run build -w @cal/web

FROM nginx:1.29-alpine
COPY apps/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/apps/web/dist /usr/share/nginx/html
EXPOSE 80
