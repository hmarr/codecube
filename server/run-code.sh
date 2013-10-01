prog=$1

if [ -z $1 ]; then
  echo "you must provide a file"
  exit 1
fi

if [ ! -f $1 ]; then
  echo "file does not exist"
  exit 1
fi

extension="${prog##*.}"
case "$extension" in
  "c")
    gcc $prog && ./a.out
    ;;
  "go")
    go run $prog
    ;;
  "py")
    python $prog
    ;;
  "rb")
    ruby $prog
    ;;
  *)
    echo "invalid language"
    exit 1
    ;;
esac

