lang=$1
prog=$2

if [ -z $1 ]; then
  echo "you must provide a language"
  exit 1
fi

if [ -z $2 ]; then
  echo "you must provide a file"
  exit 1
fi

if [ ! -f $2 ]; then
  echo "file does not exist"
  exit 1
fi

case "$lang" in
  "c")
    gcc $prog && ./a.out
    ;;
  "golang")
    go run $prog
    ;;
  "python")
    python $prog
    ;;
  "ruby")
    ruby $prog
    ;;
  *)
    echo "invalid language $lang"
    exit 1
    ;;
esac

