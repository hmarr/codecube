prog=$1
uid=$2

if [ -z $1 ]; then
  echo "you must provide a file"
  exit 1
fi

if [ -z $uid ]; then
  uid=10000
fi

echo "127.0.0.1 $(hostname)" >> /etc/hosts

groupadd code
useradd -u "$uid" -G code -d "/home/codecube" -m codecube 
chgrp code /code
chmod 0775 /code
cd /home/codecube
sudo -u codecube /bin/bash /run-code.sh $prog

