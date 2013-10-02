quota_kb=$1
uid_lower=20000
uid_upper=25000

if [ "$(id -u)" != "0" ]; then
  echo "command must be run as root :("
  exit 1
fi

if [ -z $1 ]; then
  echo "usage: $0 quota_kb"
  exit 1
fi

echo "setting quota to $quota_kb kb for uids $uid_lower to $uid_upper"
echo "press return to continue"
read confirm

for uid in $(seq $uid_lower $uid_upper); do
  setquota -u $uid 0 $quota_kb 0 0 /
done
