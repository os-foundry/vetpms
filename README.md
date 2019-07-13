# Veterinary Practition Mangement Suit (VetPMS)

[![CircleCI](https://circleci.com/gh/os-foundry/vetpms.svg?style=svg)](https://circleci.com/gh/os-foundry/vetpms)

Copyright 2019, UAB "Sonemas" (part of the OS Foundry Network)
office@osfoundry.com

## Licensing

```
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version. You may obtain a copy of the License at

    http://www.gnu.org/licenses/#AGPL

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.
```
## Local Installation

This project contains three services and uses 3rd party services such as MongoDB and Zipkin. Docker is required to run this software on your local machine.

### Go Modules

This project is using Go Module support for vendoring dependencies. We are using the `tidy` and `vendor` commands to maintain the dependencies and make sure the project can create reproducible builds. This project assumes the source code will be inside your GOPATH within the traditional location.

```
$ cd $GOPATH/src/github.com/os-foundry/vetpms
$ GO111MODULE=off go mod tidy
$ GO111MODULE=off go mod vendor
```

### Installing Docker

Docker is a critical component to managing and running this project. It kills me to just send you to the Docker installation page but it's all I got for now.

https://docs.docker.com/install/

If you are having problems installing docker reach out or jump on [Gopher Slack](http://invite.slack.golangbridge.org/) for help.

## Running The Project

All the source code, including any dependencies, have been vendored into the project. There is a single `dockerfile`and a `docker-compose` file that knows how to build and run all the services.

A `makefile` has also been provide to make building, running and testing the software easier.

### Building the project

Navigate to the root of the project and use the `makefile` to build all of the services.

```
$ cd $GOPATH/src/github.com/os-foundry/vetpms
$ make all
```

### Running the project

Navigate to the root of the project and use the `makefile` to run all of the services.

```
$ cd $GOPATH/src/github.com/os-foundry/vetpms
$ make up
```

The `make up` command will leverage Docker Compose to run all the services, including the 3rd party services. The first time to run this command, Docker will download the required images for the 3rd party services.

Default configuration is set which should be valid for most systems. Use the `docker-compose.yaml` file to configure the services differently is necessary. Email me if you have issues or questions.

### Stopping the project

You can hit <ctrl>C in the terminal window running `make up`. Once that shutdown sequence is complete, it is important to run the `make down` command.

```
$ <ctrl>C
$ make down
```

Running `make down` will properly stop and terminate the Docker Compose session.

## About The Project

The service provides record keeping for someone running a multi-family garage sale. Authenticated users can maintain a list of products for sale.

<!--The service uses the following models:-->

<!--<img src="https://raw.githubusercontent.com/os-foundry/vetpms/master/models.jpg" alt="Garage Sale Service Models" title="Garage Sale Service Models" />-->

<!--(Diagram generated with draw.io using `models.xml` file)-->

### Making Requests

#### Seeding The Database

To do anything the database needs to be defined and seeded with data. This will also create the initial user.

```
$ make seed
```

This will create a user with email `admin@example.com` and password `gophers`.

#### Authenticating

Before any authenticated requests can be sent you must acquire an auth token. Make a request using HTTP Basic auth with your email and password to get the token.

```
$ curl --user "admin@example.com:gophers" http://localhost:3000/v1/users/token
```

I suggest putting the resulting token in an environment variable like `$TOKEN`.

```
$ export TOKEN="COPY TOKEN STRING FROM LAST CALL"
```

#### Authenticated Requests

To make authenticated requests put the token in the `Authorization` header with the `Bearer ` prefix.

```
$ curl -H "Authorization: Bearer ${TOKEN}" http://localhost:3000/v1/users
```
