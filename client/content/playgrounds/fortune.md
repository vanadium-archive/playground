= yaml =
title: Playground example - Fortune
status: draft
sort: 05
= yaml =

<span style="background-color:red">
TODO(jregan): plan is to insert one or more of these as appropriate in tutorials
</span>

(Taken from: https://docs.google.com/a/google.com/document/d/189JXetSjHc980LuSl88Y7_VtG4tyZ0iggILEWD2_JKc/edit#)

This is an example of a simple Fortune application in Veyron.  It has two
parts, a client and a server.  The client can either request a fortune from the
server, or add a new fortune to the server.


## Interface Definition

The remote interface exposed by the server is defined in a .vdl file.  Here is
an example Fortune service:

    package fortune
    import "veyron2/security"

    type Fortune interface {
      // Returns a random fortune.
      Get() (Fortune string, Err error)

      // Adds a fortune to the set used by Get().
      Add(Fortune string) error
    }

The services exposes two methods - `Get` which returns a random fortune string
from a fortune repository held by the service, and `Add` that adds the provided
fortune string to the repository.


## Implementation

### Server

<div class="lang-go">
The server implements the `Get` and `Add` methods defined in the vdl interface.

Inside the `main()` function, we create a new veyron runtime with
`r := rt.Init()`, and use the runtime to create a new server with
`r.NewServer()`.

We use the `fortune` package generated from the vdl along with the fortuned
implementation to create a new fortune server.

The server listens for a tcp connection on localhost, and mounts itself on the
mounttable with the name "fortune".
</div>

<span class="lang-js">TODO(nlacasse): describe the `js` server</span>

### Client

<div class="lang-go">
The client binds to the fortune server with a `.BindFortune("fortune")` call,
and then issues a `Get` request.  We do the `.Get` request in a loop to give
the server a chance to start up.
</div>

<span class="lang-js">TODO(nlacasse): describe the js client</span>

### Code

<div class="lang-go playground" data-srcdir="/fortune/ex0-go"></div>
<div class="lang-js playground" data-srcdir="/fortune/ex0-js"></div>
