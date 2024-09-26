using Go = import "/go.capnp";

@0xa9012778dd18b453;

$Go.package("internal");
$Go.import("internal/task");

struct TaskMessage {
    tid @0 :UInt32;
    # Task ID (an integer between 0 and 4,294,967,295)

    ttype @1 :UInt8;
    # Task type (an integer between 0 and 9)

    tvalue @2 :UInt8;
    # Task value (an integer between 0 and 99)
}

interface ByteStream {
    streamTask @0 (task :TaskMessage) -> stream;
    done @1 ();
}