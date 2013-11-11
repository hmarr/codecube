FROM base
MAINTAINER Harry Marr <harry.marr@gmail.com>

RUN apt-get update
RUN apt-get install -y build-essential python ruby golang-go perl

ADD entrypoint.sh entrypoint.sh
ADD run-code.sh run-code.sh

ENTRYPOINT ["/bin/bash", "entrypoint.sh"]

