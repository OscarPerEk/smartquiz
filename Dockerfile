# Use an official image as the base (you can change this depending on your binary's dependencies)
FROM ubuntu:20.04

# Install necessary packages
RUN apt-get update && apt-get install -y wget

# Set environment variables (these can also be set in Railway's dashboard)
ENV MY_ENV_VAR=my_value
ENV SUPERKIT_ENV=production
ENV HTTP_LISTEN_ADDR=:$(PORT)
ENV DB_DRIVER=sqlite3
ENV DB_NAME=app_db

# Download your binary and environment file
RUN wget https://github.com/OscarPerEk/smartquiz/releases/download/web/app_prod_3
RUN wget https://github.com/OscarPerEk/smartquiz/releases/download/web/app_db
RUN wget https://github.com/OscarPerEk/smartquiz/releases/download/web/default.env -O ./.env

# Expose the port the binary will run on (80 in this case)
EXPOSE $(PORT)

# Define the command to run the binary
CMD ["./app_prod_3"]

