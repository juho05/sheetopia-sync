# Sheetopia Sync

Self-hosted sync server for [Sheetopia](https://github.com/juho05/sheetopia).

## Setup

### Install the Docker container

The recommended installation method for sheetopia-sync is Docker.

To get started, create a directory for sheetopia-sync:
```shell
mkdir ~/sheetopia-sync
cd ~/sheetopia-sync
```

Next, create a `docker-compose.yml` file in that directory with the following content:
```yaml
services:
  sheetopia:
    # to pin the container to a specific version replace `latest` with a version tag, e.g. `0.3.0` or `0.3`
    image: ghcr.io/juho05/sheetopia-sync:latest
    restart: unless-stopped
    volumes:
      # The data directory where sheetopia-sync stores the database and score files. To change its location
      # modify the path LEFT of the colon, e.g. "/path/to/data/dir:/data"
      - "./data:/data"
    ports:
      # The port where sheetopia-sync will be accessible. To change the port modify the value LEFT of the colon,
      # e.g. "1234:8080".
      - "8080:8080"
```

Now you can start the container:
```shell
sudo docker compose up -d
```

If everything went well you'll be able to connect to sheetopia-sync on port `8080` of your server. To verify
everything is working correctly you can try to contact the `/api/info` endpoint:
```shell
curl "http://localhost:8080/api/info"
```

A successful response will look similar to this:
```json
{"server":"sheetopia-sync","time":"2026-09-27T11:01:49.419148866Z","serverVersion":"0.3.0","apiVersion":"0.5.0"}
```

In case something does not work correctly you can view the logs with:
```shell
sudo docker compose logs -f
```

### Create a user

Before you can connect Sheetopia to your sync server, you'll need to create a user.

While the container is running execute the following command in the directory of the `docker-compose.yml` file:
```shell
# replace <name> with your desired user name
sudo docker compose exec -it sheetopia sheetopia-admin users create <name>
```

When prompted enter your desired password and the user will be created for you. You'll now be able to connect
the Sheetopia app to the server.

## License

Copyright (c) 2025-2026 Julian Hofmann

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
