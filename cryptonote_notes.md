I [previously](levin_protocol.md) described how Levin protocol header is formatted. 
The Levin body part is different, these are notes on investigating how the body part is formatted.

The body part consists of a command and its parameters. 
The command is actually identified by the "command" field in the [Levin protocol](levin_protocol.md) header.
The Levin body part then contains the fields and values for the command.
Similar to the Levin protocol main header, the body itself has its own header.
This is defined as a set of fields in CN code:

    const uint32_t PORTABLE_STORAGE_SIGNATUREA = 0x01011101;
    const uint32_t PORTABLE_STORAGE_SIGNATUREB = 0x01020101; // bender's nightmare
    const uint8_t PORTABLE_STORAGE_FORMAT_VER = 1;

The actual header of the body part is then always 010111010102010101.
This is 01011101 for PORTABLE_STORAGE_SIGNATUREA, 01020101 for PORTABLE_STORAGE_SIGNATUREB, 
and 01 for PORTABLE_STORAGE_FORMAT_VER.
Little-endian encoded it becomes 011101010101020101.

So combining this with the [Levin protocol](levin_protocol.md) header, 
the full header with levin protocol header + body-header is something like
0121010101010101150300000000000000d2070000000000000100000001000000 011101010101020101,
where the command id in the levin header in this case is d207 or 2002. The latter space-separated part is the body-header.

To parse the actual data after the headers, first the data types supported:

* Boolean
* Signed integers in sizes of 8 bits, 16 bits, 32 bits, 64 bits
* Unsigned integers in sizes of 8 bits, 16 bits, 32 bits, 64 bits
* Double precision floating points
* Objects (or more fittingly Sections)
* Strings
* Arrays of above types

These are serialized to bits in the following ways:

* Boolean: one byte, 0 for FALSE, else TRUE (so likely value 1)
* Signed/Unsigned ints: 1 byte, 2 bytes, 4 bytes, 8 bytes. Little-endian.
* Floats: 8 bytes, little-endian.
* Section: A set of name-value pairs of any of the data types (including nested Sections). Details later.
* Strings: Varint *size*, followed by *size* number of characters, one byte each.
* Arrays: Lists of a specific datatype. Details later.

# varint

So the data types mentioned something called varint. This comes up in different ways in many places of the protocol.
It stands for var-int, or I guess varying length integer. There are two different encodings the CN code uses for this.
First is the *P2P* varint, as I like to call it. The second is the *CN* varint, again, just my choise of name.
Just think of them as names, that's all.

## p2p-varint

This type of encoding packs the size of the integer in the two lowest bits of the first byte.
So the first byte is something like 0x......00. Where the last two bits define the integer size.
This is the two bits that here were set to 0 both, so the "00" ending. As it is 2 bits, it can represent 4 values:

* 0 (00): 6 bit integer - 1 byte
* 1 (01): 14 bits - 2 bytes
* 2 (10): 30 bits - 4 bytes
* 3 (11): 62 bits - 8 bytes

The actual integer value is then build by taking the associated number of bytes, 
as identified by the type id in the lowest 2 bits of the first byte.
This is a little-endian encoding, so first we read the lowest byte, then second lowest, and so on.
The resulting total bit-string is built by shifting each byte as needed, ORing them, and removing the lowest 2 bits.
This is complicated to explain, so my golang code from [p2p_parser](legacy/p2p/parser/parser_p2p.go):

    func UnpackP2PVarInt(data []byte) (uint64, int) {
        size := data[0] & 0x03
        switch size {
        case 0:
            value := data[0] >> 2
            return uint64(value), 1
        case 1:
            value := uint64(data[0])
            value |= uint64(data[1]) << 8;
            value = value >> 2
            return uint64(value), 2
        case 2:
            value := uint64(data[0])
            value |= uint64(data[1]) << 8;
            value |= uint64(data[2]) << 16;
            value |= uint64(data[3]) << 24;
            value = value >> 2
            return uint64(value), 4
        default:
            value := uint64(data[0])
            value |= uint64(data[1]) << 8;
            value |= uint64(data[2]) << 16;
            value |= uint64(data[3]) << 24;
            value |= uint64(data[4]) << 32;
            value |= uint64(data[5]) << 40;
            value |= uint64(data[6]) << 48;
            value |= uint64(data[7]) << 56;
            value = value >> 2
            return uint64(value), 8
        }
        //number of consumed bytes = second return value
    }

## cn-varint

This encoding uses the highest bit of each byte to signal whether to add another byte to the value.
The maximum size seems to be 8 bytes. 
The biggest value allowed is that expressed by *uint64*, so the value parsing is capped at that range.

Example.
Biggest value that can be represented with one byte is 7 bits = 63 in decimal.
This would be 0x01111111 in binary. After the highest bit is set, the second byte is read.
So to get the value of 64, we need two bytes.
Byte 1: 0x00000001, byte 2: 0x10000000. There you go.

Here is again my golang code from [cn_parser](legacy/p2p/parser/parser_cn.go) for this:

    func unpackCNVarInt(data []byte, valuesize int) (uint64, int) {
        var value uint64
        bytesRead := 0
    
        for i := 0; ; i++ {
            shift := i * 7
            piece := uint64(data[0])
            bytesRead++
            maxShift := valuesize * 8 - 1
            maxValue := uint64(1) << uint8(maxShift+1)
            if valuesize == 8 {
                //need this because above shift will wrap for uint64
                maxValue = 0xFFFFFFFFFFFFFFFF
            }
            rightSide := uint64(piece & 0x7f)
            rightSide = rightSide << uint8(shift)
            value = value | rightSide
            if shift > maxShift || value > maxValue {
                panic("readVarint, value overflow");
            }
            bigger := piece & 0x80
            if bigger == 0 {
                if piece == 0 && shift != 0 {
                    panic("readVarint, invalid value representation")
                }
                break
            }
            data = data[1:]
        }
    
        return value, bytesRead
    }

Now try parsing that code.. :)

# composite datatypes

So the varint explanation was just a diversion from the actual data types listed far above.
Most of the datatypes listed are rather simple and do not actually use the varints for anything.
However, when varints are used with these datatypes, they are of the *p2p* varint type.
The *cn* varint is used with the data structured contained withing these data-types in some of
the 2000+X commands only.
There are two datatypes that are composites and make more use of the *p2p* varints.
Sections and arrays. OK, maybe strings too.

## sections

Every body of every command has one high-level Section that contains all the rest of the data.
This of it like a JSON-type structure, which has a root element.
Well, you could maybe imagine some XML too, but let's not get too messed up, eh.

The Section starts with a varint value describing the number of elements under the root elements.
Think of this as the value N.
Then you loop through N values of any of the data types, including possibly other nested Sections.
For these N values, the following is performed.

The name of the variable is always first.
The name is encoded as 1 (single) byte for the length of the name (LN in following).
This is followed by LN characters, one in each byte.
So this reads 1+length of name charaters (1 for the size byte).

Once we have the name of the variable, the data of the variable directly follows.
For each variable data, the first byte tells the type of variable it is.
This contains values in the following list:

    const BIN_KV_SERIALIZE_TYPE_INT64 uint8 = 1;
    const BIN_KV_SERIALIZE_TYPE_INT32 uint8 = 2;
    const BIN_KV_SERIALIZE_TYPE_INT16 uint8 = 3;
    const BIN_KV_SERIALIZE_TYPE_INT8 uint8 = 4;
    const BIN_KV_SERIALIZE_TYPE_UINT64 uint8 = 5;
    const BIN_KV_SERIALIZE_TYPE_UINT32 uint8 = 6;
    const BIN_KV_SERIALIZE_TYPE_UINT16 uint8 = 7;
    const BIN_KV_SERIALIZE_TYPE_UINT8 uint8 = 8;
    const BIN_KV_SERIALIZE_TYPE_DOUBLE uint8 = 9;
    const BIN_KV_SERIALIZE_TYPE_STRING uint8 = 10;
    const BIN_KV_SERIALIZE_TYPE_BOOL uint8 = 11;
    const BIN_KV_SERIALIZE_TYPE_OBJECT uint8 = 12;
    const BIN_KV_SERIALIZE_TYPE_ARRAY uint8 = 13;
    const BIN_KV_SERIALIZE_FLAG_ARRAY uint8 = 0x80;

Based on the type value from this byte, the matching parser is called.
This repeats until all N variables in the Section (type code 12) are parsed.
The total number of variables in the command body can be bigger than N,
as they can contain nested Sections and Arrays.

## arrays

From the above list of data types, you may have noticed there are two array related values for data types:

    const BIN_KV_SERIALIZE_TYPE_ARRAY uint8 = 13;
    const BIN_KV_SERIALIZE_FLAG_ARRAY uint8 = 0x80;

I don't recall seeing the BIN_KV_SERIALIZE_TYPE_ARRAY actually used, and reading the code it might not even work.
But if it does, they both point to the same underlying implementation for parsing once you know its an array
you are looking at.
So BIN_KV_SERIALIZE_FLAG_ARRAY is used.
The way it is used is the value 0x80 maps to binary of 0x10000000.
So the highest bit is used to signal if the value is an array of some type.
The remaining 7 lowest bits then define the type itself, which can be any of the above values from 1-12
(well 13 too in theory).
To see if it is an array, first test the highest bit, then remove that bit and check the remaining value for 
the type of values in the array.

After the type byte, the first value parsed is a p2p-varint defining how many values are in the array.
Each value is then read as the data structure for that type defines.
So 1 byte for uint8, 2 for uint16, and so on. This repeats for the size of the array.

## strings

Strings start with a p2p-varint for the number of characters in the string.
This is followed by a matching number of bytes, one byte for a character.
Much of the CN code actually seems to use this as a form of byte-array instead of readable text.

# commands

The above is the general structure of the command body data.
Each command has its own set of variables encoded using this format,
and some have their own special data formats embedded into this.
Note that I have not re-implemented a fully working daemon from this at this point, so this may not all be 100% correct.
Most of the ones presented below parse fine for the packets I captured from the CN daemon though,
and they also produce reasonable results.
2003 is the exception here, as I never finished reversing it in full. 
So, examples follow.

## P2P or 1000+X commands

For some reason, the CN code seems to call  this set P2P commands. So I go with that.

### 1001: Handshake

Handshake seems to be what two nodes use to say hello when connecting. 
Here is an example dump of the packet data, showing the TCP payload:

    0121 0101 0101 0101 bb00 0000 0000 0000  .!..............
    01e9 0300 0000 0000 0001 0000 0001 0000  ................
    0001 1101 0101 0102 0101 0809 6e6f 6465  ............node
    5f64 6174 610c 140a 6e65 7477 6f72 6b5f  _data...network_
    6964 0a40 .... .... .... .... .... ....  id..............
    .... .... 0776 6572 7369 6f6e 0801 0770  ..C..version...p
    6565 725f 6964 05.. .... .... .... ..0a  eer_id..........
    6c6f 6361 6c5f 7469 6d65 0538 0145 5b00  local_time.8.E[.
    0000 0007 6d79 5f70 6f72 7406 792e 0000  ....my_port.y...
    0c70 6179 6c6f 6164 5f64 6174 610c 080e  .payload_data...
    6375 7272 656e 745f 6865 6967 6874 0634  current_height.4
    4c09 0006 746f 705f 6964 0a80 a8ce 341b  L...top_id....4.
    032a fcf3 2c50 57cd ab66 d0e0 5be4 13f5  .*..,PW..f..[...
    a2e9 e4d3 883b ec5a e310 a5ab            .....;.Z....

So looking back at the Levin protocol description, it starts with the Levin header,
followed by "0xbb" for body data size of 187 bytes. 01 follows for requiring a return value.
e903 = 03e9 = 1001 = the command id. This follows by the Levin request flag and protocol versions of 01.
The body header is "1101 0101 0102 0101", so the actual body data starts after this.

First it starts with 0809. This is the start of the root Section element.
08 is a varint packed value for number of elements in this Section.
08 translates to binary "00001000".
Lowest two bits are 0, marking it as a single byte integer.
Shifted right by two to remove the size bits, it becomes "00000010". 
This is binary for 2, so this Section has 2 elements.

The first Section element follows.
09 is for 9 character name that follows for the first item, "node_data".
0c (12) is the type of the first item, Section again.
Section size is 0x14 = 20 = 00010100 = 00000101 = 5.
So inner section has 5 elements.
First element has a name of 10 characters, "network_id".
This is of type 0a = 10 = string.
String size is 0x40 = 64 = 01000000 = 00010000 = 16.
The string content is the set of data I blanked in the above dump as "...." values.
I have no idea what it actually contains, but seems to be some form of binary dump.
The implementation is hiding somewhere in the CN code, this is just an analysis of the protocol format.

The second field of the inner Section is "version". 
It starts with 07 for name length of "version", followed by the characters.
This has type 0x08 = BIN_KV_SERIALIZE_TYPE_UINT8. So it is uint8 for version.
The version value is a single by of 0x01.
After this starts again next field value for field "peer_id" for the inner Section.
This is type 05, so uint64. 
Following 8 bytes are for the peer_id value, which, again, I don't know how it is formed.

This goes on until the 5th (and last) element of the inner Section "my_port".
This follows by "payload_data", the second outer element of the root Section.
So the actual structure of the data for this command is:

Root Section
* node_data: Section
    * network_id: uint8 "string", 16 bytes
    * version: uint8, 1 byte
    * peer_id: uint64, 8 bytes
    * local_time: uint64, 8 bytes, unix timestamp in seconds since Unix epoch.
    * my_port: uint32, 4 bytes, TurtleCoin defaults to 11897 at this time.
* payload_data: Section
    * current_height: uint32, 4 bytes. Block height at which the blockchain is at.
    * top_id: uint8 "string", 32 bytes (in above dump at least), seems to be hex-encoded string value of top block in the chain.

I don't have the details on how each the fields is constructured, so just take the above as the protocol format with some notes.
However, I can look at the values a bit.

The "current_height" in the above dump is 609332.
The "top_id", when printed out as a hex-string is a8ce341b032afcf32c5057cdab66d0e05be413f5a2e9e4d3883bec5ae310a5ab.
With a [TurtleCoin block explorer](https://turtle.land), I can see how this matches.
The block id 609332 does not match the top_id value.
However, block id 609331 has the exact hash string of top_id above.
So it seems that current_height indicates which block is next, 
and the top_id is the hex-encoded hash of the last valid block on the chain.

Similarly, local_time provides a valid timestamp when input into an [epoch converter](https://www.epochconverter.com/).
And the port number of 11897 can be found in the TurtleCoin source code. So it all at least seems to parse fine for this one.

### 1001: Handshake reply

When a node sends a handshake to another node, it gets back a reply with the other nodes info.
I believe this is the response expected flag in the Levin protocol header.
In the handshake case, this is exactly the same data set, but also with a list of known peer nodes for the other end.
So structure:

Root Section
* node_data: Section
    * network_id: uint8 "string", 16 bytes
    * version: uint8, 1 byte
    * peer_id: uint64, 8 bytes
    * local_time: uint64, 8 bytes, unix timestamp in seconds since Unix epoch.
    * my_port: uint32, 4 bytes, TurtleCoin defaults to 11897 at this time.
* payload_data: Section
    * current_height: uint32, 4 bytes. Block height at which the blockchain is at.
    * top_id: uint8 "string", 32 bytes (in above dump at least), seems to be hex-encoded string value of top block in the chain.
* peer_list: uint8 "string", custom formatted list of peer node information

So the other information is not so interesting, as it was covered already for handshake parsing. 
But the peer_list is new and has a custom format embeded in the binary string.
In general, it seems the "string" datatype just refers to binary format "strings", so just byte arrays basically.
The peer_list parses as a set of peer information structures, each one containing:

* ip address: 4 single byte uint8 values, each one octet of an ipv4 address
* port number: 4 byte uint32 value, little-endian
* peer id type: 8 bytes uint64 value, little-endian
* last seen: 8 bytes uint64 value, little-endian unix epoch timestamp

This simply a form of a byte array dumped directly from memory.
So to parse this, first count the size of a single data structure.
This is 4+4+8+8 = 24 bytes.
In this case, the data from the response packet dump has 2496 bytes in the peer_list byte string.
Divided by 24 this becomes 104 peer list entries.
So, simply looping the peer_list data in sections of 24 bytes gives a list of peers.
Here is the code from my golang [parser](legacy/p2p/parser/parser_p2p.go):

    func parsePeerList(data []uint8) []PeerInfo {
        count := len(data)/24
        peerlist := []PeerInfo{} //todo: set capacity to count
        for i := 0 ; i < count ; i++ {
            ipBytes := []uint8{data[0], data[1], data[2], data[3]}
            ip1 := strconv.FormatUint(uint64(data[0]), 10)
            ip2 := strconv.FormatUint(uint64(data[1]), 10)
            ip3 := strconv.FormatUint(uint64(data[2]), 10)
            ip4 := strconv.FormatUint(uint64(data[3]), 10)
            ipStr := ip1 + "." + ip2 + "." + ip3 + "." + ip4
            port := binary.LittleEndian.Uint32(data[4:8])
            peerIdType := binary.LittleEndian.Uint64(data[8:16])
            lastSeen := binary.LittleEndian.Uint64(data[16:24])
            pi := PeerInfo{peerIdType, lastSeen, ipStr, ipBytes, port}
            peerlist = append(peerlist, pi)
            data = data[24:]
        }
        return peerlist
    }

Running this and printing the results provides some valid looking data, with at least seeming valid ip addresses and timestamps.
I cannot say about peer id's, since I have not looked in too deep into how they are constructed.
Interestingly, the timestamps show quite a bit of fluctuation, and there were none of the hard-coded seed node ip's
in the list I got. Still, it seems quite valid, but not copying it here anyways.
The list actually also contains lots of 0.0.0.0 entries towards the end. 
So it might be that the list is always of this specified length, even if at some point there are not that many peers
connected to a particular node.
Certainly, how all the peer id's are constructed, how peers are selected, and so on would be interesting to see.


### 1002: Timed Sync

Timed sync seems to be a way for nodes to exchange synchronization information after a handshake.
Again, no idea really, just guessing, would need a lot more reading of the CN code to really say.
But the structure is very similar to that of the handshake:

Root section
* payload_data: Section
    * current_height: uint32, 4 bytes. Block height at which the blockchain is at.
    * top_id: uint8 "string", 32 bytes (in above dump at least), seems to be hex-encoded string value of top block in the chain.

This was from a packet capture I analyzed, I am guessing this is the request part, and there is likely a response
part with possibly added information. In fact, CN code defines the response structure as:

    struct response
    {
        uint64_t local_time;
        CORE_SYNC_DATA payload_data;
        std::list<PeerlistEntry> local_peerlist;
    
        void serialize(ISerializer &s)
        {
            KV_MEMBER(local_time)
            KV_MEMBER(payload_data)
            serializeAsBinary(local_peerlist, "local_peerlist", s);
        }
    };

So my guess is, this is just a similar way to synch the peer list, with same data structure as handshake,
just missing information on the node itself. 
Likely the information on node itself is assumed to stay static after handshake, which makes sense.
The blockchain moves, the peerlist has connections added and lost.
Nodes own address, port, etc. stays.

### 1003: Ping

Ping seems to be just a way for nodes to check if a connected node is still alive.
The request has no information (body variables) in it, just the command ID in the Levin header.
You might still want to parse the packet to see it is valid, but other than that checking the command id and request type seems enough.
Again, this is what my golang [parser](legacy/p2p/parser/parser_p2p.go) for this command does.

The response packet I did not capture and parse but from the CN code, we can see it has a few basic fields:

    #define PING_OK_RESPONSE_STATUS_TEXT "OK"

    struct request
    {
        /*actually we don't need to send any real data*/
        void serialize(ISerializer &s)
        {}
    };

    struct response
    {
        std::string status;
        PeerIdType peer_id;

        void serialize(ISerializer &s)
        {
            KV_MEMBER(status)
            KV_MEMBER(peer_id)
        }
    };

So, the request is empty bodied (just root Section), while the response has two values. Something like this:

Root section
* status: uint8 "string", 2 bytes (for "OK)
* peer_id: uint64, 8 bytes

Note that response structure is a guess right now, did not go looking for separate capture on it, I just analyzed the request part.
But should be relatively straightforward to capture it and test.

## CN or 2000+X commands

### 2001: Notify New Block

Never tried to parse this, after 2003 gave me enough of a headache, I left it for now.

### 2002: Notify New Transactions

This seems to be a message to synchronize new transactions between nodes.
The structure seems to be a bit of a mess, but I guess it works:

Root section
* txs: array of uint8 "string" types, in practice just one value in the array (of uint8 type, code 10 from type list) 
    * uint8 "string", 780 bytes in the packet dump I used, might vary in general. custom formatted transaction list.

There is a single array element under the root section. This is created using the 0x80 array mask, with
the masked type as BIN_KV_SERIALIZE_TYPE_STRING uint8 (= 10).
It seems like someone had the right idea of putting in an array of transaction objects.
However, it seems all the CN (2000+X) commands are using the "P2P" style general headers for the body,
but then dumping any specific data into a byte array represented by the BIN_KV_SERIALIZE_TYPE_STRING type.
The contents of the "txs" array are actually a single element, containing the actual array content custom packed.
This is similar to *peer_list* in the *1001 Handshake reply* message.

I would expect this is how all the 2000+X are formatted then.
Also, this is where the "CN" style varints are now used. 
So in parsing the byte array for the "txs", all varints are now of the second "CN" type encoding.
This is the structure of transactions in that byte array:

* version: uint8, 1 byte
* unlock_time: uint64, 8 bytes
* input_count: uint64, 8 bytes
* list_of_inputs: input_count times the following:
    * type_tag: uint8, 1 byte
    * amount: uint64, 8 bytes
    * output_count: uint64, 8 bytes
    * list_of_output_indices for this input, output_count times the following:
        * output_index: uint32, 4 bytes
* output_count: uint64, 8 bytes
* list_of_outputs: output_count times the following:
    * amount: uint64, 8 bytes
    * type_tag: uint8, 1 byte
    * public_key: uint8\[32\], 32 bytes. public key associated with output.
* extra_size: uint64, 8 bytes
* tx_extra: uint8\[extra_size\], transaction extra, whatever someone decides to dump in there. the standard had some suggestions, in practice people seem to have dumped their stories etc. in this.
* signatures, input_count times the following:
    * for each input_x in list_of_inputs:
        * for each output_index in input_x:
            * signature: uint8\[64\], 64 bytes, signature matching the public key of tx_in\[tx_out\]

All uint8, uint32, uint64 in this transaction structure are packed using the "CN" varint encoding format.

I could go into lot more detail about these different fields inside the transaction, 
but the [CryptoNode Standard for transaction](https://cryptonote.org/cns/cns004.txt) 
seems to have reasonable descriptions of the fields.
Note that the standard discusses something called "transaction prefix",
which I merged into the transaction structure above.
Mostly because I did not see a need to have separate sub-structures as its not that complicated,
and its not repeating or anything.
However, the CN code implementation also seems to mirror this terminology and struct objects.

Generally, the [CryptoNode Standards](https://cryptonote.org/standards/) seem quite helpful in understanding
the fields, they just don't quite detail the information enough to be able to parse everything from the protocol.
Or maybe if I read all the standars in detail it could be inferred to some extent.

In any case, this array of custom formatted transaction at least parses fine with my [goland code](legacy/p2p/commands/cryptonote_commands.go).
Note that the packet dump I used only had a single entry in the top level array, in other cases there might be more.
But they should at least parse with the structure/code above.

Interestingly, looking at the packet dumps, the same message kept repeating with the exact same content for the period
I ran tcpdump on this.
Might be because the node was synchronizing, it was re-resending this as new transaction until the other node synchs up to it?
Seems a bit of a waste, but if it works I guess?
The CN code also maintains a status flag telling whether it has finished synchronizing or not, and discards these
messages if it is state such as synchronizing still.
I actually removed this check for the duration of tracing how transactions are serialized,
which might be useful.
This was in CryptoNoteProtocolHandler::handle_notify_new_transactions, the check for 

    if (context.m_state != CryptoNoteConnectionContext::state_normal) {

### 2003: Notify Request Get New Objects

After finally getting 2002 to parse, I also had a look at 2003. 
Unfortunately, I did not quite figure out this one so far and it gets pretty frustrating to read the CN code after a while.
But here are a few notes.

The packet dump I was looking at for this has a single field in its body:

* blocks: uint8 "string", 3200 bytes in the packet dump I used

My guess is, this has to be another custom-formatted byte array with information on blocks.
Since top_id contains block id's as 32 byte arrays in other messages, it might fit.
Calculating number of blocks in this way gives 3200/32 = 100 blocks in a single message.
Chunking this into 32 byte parts, and using each as a hex-encoded byte array to represent block id's seems
to produce quite reasonable results.
Towards the end, the packet I had, had a number of zeroed id's, which also seemed to match the other
types of message with similar arrays (e.g., peer_list).
However, it does not quite match as the last non-zero one has a few bytes of non-zero and rest are zero.
Also, taking the generated hashes, I could not find them on the block-explorer. So maybe not quite right.

Looking at the CN code, the following is the definition for this (request):

    struct NOTIFY_REQUEST_GET_OBJECTS_request
    {
        std::vector<Crypto::Hash> txs;
        std::vector<Crypto::Hash> blocks;

        void serialize(ISerializer &s)
        {
            serializeAsBinary(txs, "txs", s);
            serializeAsBinary(blocks, "blocks", s);
        }
    };

So it seems to be serializing a list of hashes as binary lists.
The Crypto::Hash type is defined as: 

    struct Hash {
      uint8_t data[32];
    };

So this also supports the idea of a 32 byte split.

But the code also lists "txs", and this was not in the packet? Perhaps if the vector is empty, 
it is ignored and not serialized at all? Don't know.
The code might also input some size values first into the binary stream for the vector.
Again, don't know.
The odd thing was, I could not quite find where the data is encoded or decoded into binary.
My traces showed it running various parts of what the CN code defines as "JSON" serializer,
but typically this (in other commands) leads further into the binary serializer.
Did not quite do it here, and I got tired of it.

In any case, below I will put some notes on how I tried to trace it if someone wanders here and is interested to investigate more.

# tracing the CN code

The CN code has various trace levels to print information on what it does.
However, many times this does not have the exact trace of interest for this type/level of tracing I was doing.
The most detailed trace level will also produce pretty huge logs when running the daemon. 
So I used my custom traces, which are just prints in various parts of the code.
Here is how I implemented it:

Somewhere inside the code, declare a globally accessible flag variable. I put this in CryptoNoteProtocolHandler.h:

    namespace CryptoNote
    {
        extern int leturt;

And matching setting in CryptoNoteProtocolHandler.cpp:

    namespace CryptoNote
    {
        int leturt = 1;

Then just littered the codebase with prints like this:

    if (CryptoNote::leturt) std::cout<<"HELLO WORLD\n";

And more specifically, adding more information on the context, such as:

    bool JsonInputValueSerializer::binary(void *value, size_t size, Common::StringView name)
    {
        std::string strName(name);
        if (CryptoNote::leturt) std::cout<<"JSI-SERIALIZE 18:"<<strName<<"\n";

Most of the serialization related stuff seems to happen inside the code in the *Serialization* directory.
So I found a good place to instrument code was to instrument all of the XXInputStreamSerializer and XXOutputStreamSerializer code, 
meaning print out every call into any method there, with information of interest.
Similarly, SerializationTools.h, SerializationOverrides.h, and ISerializer.h seemd like good places to trace.

However, since there is lots and lots of serialization happening, and I was generally interested only in trying to
reverse a single command/message at a time, the logs from this type of trace quickly grow huge, and it is difficult
to find out where the actual serialization of interest is happening.
So setting the flag to only print when this serialization should happen can help.
First set if to zero globally:

    namespace CryptoNote
    {
        int leturt = 0;

Now find places to set it to true in.

*CryptoNoteProtocolHandler::handleCommand* in CryptoNoteProtocolHandler.cpp 
sets up all the 2000+X command handlers, so one good place to start.
Just above this is the definition of *notifyAdaptor()*, which is where the commands seem to pass through.
So in this we can do some tricks such as:

        typedef typename Command::request Request;
        int command = Command::ID;
        if (command == 2002) {
            CryptoNote::leturt = 1;
        }
        ...
        if (!LevinProtocol::decode(reqBuf, req))
        ...
        CryptoNote::leturt = 0;

Since this calls LevinProtocol::decode, that is another option:

        Common::MemoryInputStream stream(buf.data(), buf.size());
        KVBinaryInputStreamSerializer serializer(stream);
        //could also set the trace flag here based on some magic parameters
        std::cout<<"levin decoding value type:"<<typeid(value).name()<<"\n";
        serialize(value, serializer);
        std::cout<<"levin finished decoding value type:"<<typeid(value).name()<<"\n";

The typeid() is a magic trick I found from StackOverflow for printing out concrete type of passed argument.
This can be useful to see which concrete object is called, since the CN code overuses abstractions and stuff.
For example, sometimes the serializer is difficult to figure out which one it is.

Another option is tracing from the handler methods called from *CryptoNoteProtocolHandler::handleCommand*.
For example, CryptoNoteProtocolHandler::handle_request_get_objects for NOTIFY_REQUEST_GET_OBJECTS.
Start the trace on call, end when done:

    int CryptoNoteProtocolHandler::handle_request_get_objects() 
        CryptoNote::leturt = 1;
        ...
        CryptoNote::leturt = 0;
        return 1;

Sometimes, when it gets really frustrating to figure out, I would also just set the flag globally and
do searches for specific prints in the log file. But it gets yuuuuge fast.

There are also places in the code that seem to do nothing, yet it seems like the CN developers tried their
best to obfuscate what it does. Operator overloading is all over the place in the Serialization code.
And things like this:

    KVBinaryInputStreamSerializer::KVBinaryInputStreamSerializer(Common::IInputStream &strm) : JsonInputValueSerializer(
            parseBinary(strm))
    {
    }

Does nothing, right? Except the parameter passed to the superclass constructor JsonInputValueSerializer() 
is actually a function call to parseBinary(strm), which actually does a lot of stuff. Brilliant.

I am no c++ master in any way, so maybe it is all wrong, but seemed to work decent most of the time.
Of course, sometimes I was just left wondering what is actually happening since the trace was not always
showing what I was looking for, and could not figure where else it could be happening.
Most recently this was with the 2003 message/command.

Running the whole deamon from within a debugger would be great of course,
but the CN code seems like such as mess to me, I had no idea even how to set it up as a debuggable project in CLion.
Just litter everything with prints, move the flag around and see what happens. 
Run make (which sometimes takes a looong time, so good to minimize changes by using a flag or something).

If you figure out more of the structures, what the messages are used for, how they form the overall protocol interactions, etc.
I am interested to hear.

Cowabunga, dudes :)
