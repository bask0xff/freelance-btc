FROM node:20-alpine
WORKDIR /app

# Немного увеличиваем терпимость npm к сетевым задержкам/обрывам — на Docker Desktop под Windows
# запросы к registry.npmjs.org иногда подвисают/таймаутят даже при рабочем интернете в контейнере.
RUN npm config set fetch-timeout 300000 \
    && npm config set fetch-retries 5 \
    && npm config set fetch-retry-mintimeout 20000 \
    && npm config set fetch-retry-maxtimeout 120000

COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
