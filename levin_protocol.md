#Levin protocol

TurtleCoin daemon seems to talk using Levin Protocol. This is just a look at the protocol..

The protocol has a header and body part.

##header

The header is defined in LevinProtocol.cpp as:

    struct bucket_head2
    {
        uint64_t m_signature;
        uint64_t m_cb;
        bool m_have_to_return_data;
        uint32_t m_command;
        int32_t m_return_code;
        uint32_t m_flags;
        uint32_t m_protocol_version;
    };

*m_signature* is always set as

    const uint64_t LEVIN_SIGNATURE = 0x0101010101012101LL;  //Bender's nightmare

*m_cb* defines the size of the body part of the message.

*m_have_to_return_data* defines whether a message is a notification or needs a reply/response data.

*m_command* defines a protocol command. Such as COMMAND_PING, COMMAND_REQUEST_NETWORK_STATE, ... as defined
in P2pProtocolDefinitions.h.

*m_return_code* any return code for request.

*m_flags* protocol flags, mainly seems to define if the message is request or reply. LEVIN_PACKET_RESPONSE vs
LEVIN_PACKET_REQUEST in LevinProtocol.cpp.

*m_protocol_version* the protocol version, likely for some updates and compatibility checks.

##body

The body part is simply binary data written over the network as is.

##serialization

The writeStrict() and readStrict() functions in LevinProtocol.cpp seem to push the data into a binary format
using some default c++ operations. What does this mean, and how shoudl serialization be written?

Here is an example dump of data, captured with tcpdump and visualized with Wireshark:

    0000   12 34 56 78 9a bc .. .. .. .. .. .. .. .. .. ..   .4Vx.¼...&......
    0010   .. .. .. .. .. .. .. .. .. .. .. .. .. .. .. ..   ................
    0020   .. .. .. .. .. .. .. .. .. .. .. .. .. .. .. ..   ................
    0030   .. .. .. .. .. .. 01 21 01 01 01 01 01 01 15 03   .......!........
    0040   00 00 00 00 00 00 00 d2 07 00 00 00 00 00 00 01   .......Ò........
    0050   00 00 00 01 00 00 00 01 11 01 01 01 01 02 01 01   ................
    0060   04 03 74 78 73 8a 04 0d 0c 01 00 01 02 f0 2e 08   ..txs........ð..
    0070   a0 ee 23 d5 47 01 01 01 01 01 01 f3 2c 91 b7 54    î#ÕG......ó,.·T

Up and until "01 21" it is all network protocol headers. 
The first part "12 34 56 78 9a bc" is the Ethernet destination address.
This is followed by various other Ethernet and TCP/IP headers, all which I blanked out in there since they
are not important for the Levin protocol.
However, I initially did spent some good time reading the code and looking through trying to figure out 
where in the code "123456789abc" is defined. News flash, nowhere, if I just had looked at Wireshark in more
detail it actually shows me these are the network protocol headers all the way until "01 21".

So from "01 21" forward it is the actual payload data. What is it?

###signature

The protocol header should start with the signature value of 0x0101010101012101LL.
LL in the end defines it as "long long" value, which i guess just makes it suitable for uint64 in c++.
uint 64 is 8 bytes in memory, and first 8 bytes of the payload are
"01 21 01 01 01 01 01 01", which seems a bit like the signature but not quite. 
My original question to investigate this was to see if I should encode the values in the header as
little-endian or big-endian.

To see what I get, I tried in my golang implementation to run both litle-endian and big-endian on the header signature.
The big-endian encoding is "01 01 01 01 01 01 21 01", which matches the actual signature from the code.
The little-endian encoding gives "01 21 01 01 01 01 01 01", which matches the tcpdump packet data.
So the header should be encoded as little-endian, and this is actual Levin protocol formatted data.

###body size

Following *m_signature* is *m_cb*, which is the protocol body size.
This is again uint64 (somebody expects pretty big data in a single msg..), and the next 8 bytes of the dump are
"15 03 00 00 00 00 00 00".
Since all data in the protocol implementation is serialized in the same way, I expect it all to be little-endian.
So interpreting this as little-endian, I get 0x0000000000000315. 
Convering this to decimal I get 789. 
Counting the header bytes together from the "bucket_head2", I get 33 bytes (8 + 8 + 1 + 4 + 4 + 4 + 4).
Looking at the Wireshark data for the network protocol headers, the TCP packet size header is 822.
Which matches exactly 789 + 33 = 822.

So the serialization writes the header first, immediately followed by the body data.

###have to return data?

Next in the header is *m_have_to_return_data*.
In this dump it is 00, which translates to false. So this should be a notification?

###command

The command value is next, which here is "d2 07 00 00" -> 0x07d2 -> 2002.
To figure what this is, it is necessary to find what commands are defined in the code.
The first and maybe more obvious one is in P2pProtocolDefinitions.h, which defines a set of commands such
as ping, and uses a base value of 1000.
2002 matches a definition in CryptoNoteProtocolDefinitions.h, which uses a base value of 2000.
2002 is then NOTIFY_NEW_TRANSACTIONS, or so I think. 
This also matches the boolean value of *m_have_to_return_data* above, meaning it is intended to be a notification.

###return code

Return code in this case is "00 00 00 00", 
which is likely just null value since a notification has no reply/return value expected.

###flags

Flags in this dump are "01 00 00 00", which translates to 0x00000001. 
This matches the LevinProtocol.cpp definition of LEVIN_PACKET_REQUEST = 0x00000001.
So it is a notification but the flag types are request and reply, and this is just a request with no reply expected.

###protocol version

Protocol version is a similar set of 4 bytes "01 00 00 00". So version 1 (little-endian).

##body

The body data in this case seems to start from "01 11 01 01 01 01 02 01 01", whatever this is.
Likely some transaction data, since this was a message with command NOTIFY_NEW_TRANSACTIONS.
Would need to find this command data format from CryptoNoteProtocolDefinitions.cpp to see, 
but the general Levin protocol seems to be like this.














