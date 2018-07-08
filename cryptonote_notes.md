I previously described how levin protocol header is formatted. The levin body part is different, this is notes on investigating how the body part is formatted.

To serialize the command data, LevinProtocol.h calls 
    void KVBinaryOutputStreamSerializer::dump(IOutputStream &target)

which writes the data as

        KVBinaryStorageBlockHeader hdr;
        hdr.m_signature_a = PORTABLE_STORAGE_SIGNATUREA;
        hdr.m_signature_b = PORTABLE_STORAGE_SIGNATUREB;
        hdr.m_ver = PORTABLE_STORAGE_FORMAT_VER;

        Common::write(target, &hdr, sizeof(hdr));
        writeArraySize(target, m_stack.front().count);
        write(target, stream().data(), stream().size());

So first it writes the body header part. The body header signatures are defined as

    const uint32_t PORTABLE_STORAGE_SIGNATUREA = 0x01011101;
    const uint32_t PORTABLE_STORAGE_SIGNATUREB = 0x01020101; // bender's nightmare
    const uint8_t PORTABLE_STORAGE_FORMAT_VER = 1;

The actual header of the body part is then always 010111010102010101, 
which is 01011101 for PORTABLE_STORAGE_SIGNATUREA, 01020101 for PORTABLE_STORAGE_SIGNATUREB, and 01 for PORTABLE_STORAGE_FORMAT_VER
when little-endian encoded it becomes 011101010101020101.

So the full header with levin protocol header + body-header is
0121010101010101150300000000000000d2070000000000000100000001000000 011101010101020101,
where the command id in the levin header in this case is d207 or 2002. The latter space-separated part is the body-header.

The code seems to use key-value pairs for the body data, thus it is referenced as KV. So the body-header is called KV-header in the code.

After body header is written, writeArraySize writes the size of m_stack into the serialization stream. This is done as

        if (val <= 63)
        {
            return packVarint<uint8_t>(s, PORTABLE_RAW_SIZE_MARK_BYTE, val);
        } else if (val <= 16383)
        {
            return packVarint<uint16_t>(s, PORTABLE_RAW_SIZE_MARK_WORD, val);
        } else if (val <= 1073741823)
        {
            return packVarint<uint32_t>(s, PORTABLE_RAW_SIZE_MARK_DWORD, val);
        } else
        {
            if (val > 4611686018427387903)
            {
                throw std::runtime_error("failed to pack varint - too big amount");
            }
            return packVarint<uint64_t>(s, PORTABLE_RAW_SIZE_MARK_INT64, val);
        }

 here packVarint formats the array size as

     size_t packVarint(IOutputStream &s, uint8_t type_or, size_t pv)
    {
        T v = static_cast<T>(pv << 2);  //shift left by two to fit the two size bits in
        v |= type_or;                   //OR the two lowest bits in to represent the byte count
        write(s, &v, sizeof(T));        //write sizeof(T) bytes from address &v into stream
        return sizeof(T);
    }

the size of the array (the *pv* parameter) is first shifted left by two bits. For this reason, max size of 63 is used for packing the size into a single byte. This is because the largest value that can be expressed with 6 bits is 63 (binary 111111). And 63 would be encoded as 11111100, since it was shifted left by two. The size id bits are 

    const uint8_t PORTABLE_RAW_SIZE_MARK_BYTE = 0;
    const uint8_t PORTABLE_RAW_SIZE_MARK_WORD = 1;
    const uint8_t PORTABLE_RAW_SIZE_MARK_DWORD = 2;
    const uint8_t PORTABLE_RAW_SIZE_MARK_INT64 = 3;

These are the four values that can be expressed by the two lowest bits. Decoding will know how many bytes to take based on the lowest two bits of the first byte in size.

The next question is, what is after array size. This is the 

        write(target, stream().data(), stream().size());

in the dump() method after writeArraySize().

What is stream() it is MemoryStream &stream();. What is MemoryStream? Its in src/Serialization/Memorystream.h/cpp in TC code. Which in practice is just a front for std::vector<uint8_t> m_buffer;. So the contents of this vector are written after the array size.

The stream() method is 

    MemoryStream &KVBinaryOutputStreamSerializer::stream()
    {
        assert(m_objectsStack.size());
        return m_objectsStack.back();
    }

So all the data written is actually from m_objectsStack variable. Where is this written to?

This is written in

    bool KVBinaryOutputStreamSerializer::beginObject(Common::StringView name)
    {
        checkArrayPreamble(BIN_KV_SERIALIZE_TYPE_OBJECT);

        m_stack.push_back(Level(name));
        m_objectsStack.push_back(MemoryStream());

        return true;
    }

    void KVBinaryOutputStreamSerializer::endObject()
    {
        assert(m_objectsStack.size());

        auto level = std::move(m_stack.back());
        m_stack.pop_back();

        auto objStream = std::move(m_objectsStack.back());
        m_objectsStack.pop_back();

        auto &out = stream();

        writeElementPrefix(BIN_KV_SERIALIZE_TYPE_OBJECT, level.name);

        writeArraySize(out, level.count);
        write(out, objStream.data(), objStream.size());
    }

Which is plenty of c++ specific vector operations, likely called from all over the place. 

Needs someone with more c++ skills to go into all that. What else can we look into?

Besides serialization, there has to be de-serialization. Maybe that is simpler?

De-serialization is done in KVBinaryInputStreamSerializer.h/cpp.

The

     size_t readVarint(Common::IInputStream &s)

method confirms the formatting of size variables.

The

    JsonValue loadValue(Common::IInputStream &stream, uint8_t type)

method shows how a number of different data types is parsed.

The 

    JsonValue parseBinary(Common::IInputStream &stream)

method seems to be where its at. So it starts by reading the header

        auto hdr = readPod<KVBinaryStorageBlockHeader>(stream);

        if (
                hdr.m_signature_a != PORTABLE_STORAGE_SIGNATUREA ||
                hdr.m_signature_b != PORTABLE_STORAGE_SIGNATUREB)
        {
            throw std::runtime_error("Invalid binary storage signature");
        }

        if (hdr.m_ver != PORTABLE_STORAGE_FORMAT_VER)
        {
            throw std::runtime_error("Unknown binary storage format version");
        }

This follows by 

        return loadSection(stream);

which does this

        JsonValue sec(JsonValue::OBJECT);
        size_t count = readVarint(stream);
        std::string name;

        while (count--)
        {
            readName(stream, name);
            sec.insert(name, loadEntry(stream));
        }

So it reads a JSON object, which has *count* number of values in it. These are loaded with readName() and loadEntry(). This does

    void readName(Common::IInputStream &s, std::string &name)
    {
        uint8_t len = readPod<uint8_t>(s);
        if (len)
        {
            name.resize(len);
            read(s, &name[0], len);
        }
    }

Where readPod reads data from the stream of the given size. So in this case it reads one byte to get the name length. This is followed by reading the number of bytes matching the name length. So the format is <single byte for size><name in number of bytes from first byte>.

To load entry for the name:

    JsonValue loadEntry(Common::IInputStream &stream)
    {
        uint8_t type = readPod<uint8_t>(stream);

        if (type & BIN_KV_SERIALIZE_FLAG_ARRAY)
        {
            type &= ~BIN_KV_SERIALIZE_FLAG_ARRAY;
            return loadArray(stream, type);
        }

        return loadValue(stream, type);
    }

So start with reading a single byte to define the data type. If the first byte says array, call loadArray, else call loadValue.

Arrays:

    JsonValue loadArray(Common::IInputStream &stream, uint8_t itemType)
    {
        JsonValue arr(JsonValue::ARRAY);
        size_t count = readVarint(stream);

        while (count--)
        {
            arr.pushBack(loadValue(stream, itemType));
        }

        return arr;
    }

So just read number of values in array first as the varint type. The loadValue() as with plain values.

    JsonValue loadValue(Common::IInputStream &stream, uint8_t type)
    {
        switch (type)
        {
            case BIN_KV_SERIALIZE_TYPE_INT64:
                return readIntegerJson<int64_t>(stream);
            case BIN_KV_SERIALIZE_TYPE_INT32:
                return readIntegerJson<int32_t>(stream);
            case BIN_KV_SERIALIZE_TYPE_INT16:
                return readIntegerJson<int16_t>(stream);
            case BIN_KV_SERIALIZE_TYPE_INT8:
                return readIntegerJson<int8_t>(stream);
            case BIN_KV_SERIALIZE_TYPE_UINT64:
                return readIntegerJson<uint64_t>(stream);
            case BIN_KV_SERIALIZE_TYPE_UINT32:
                return readIntegerJson<uint32_t>(stream);
            case BIN_KV_SERIALIZE_TYPE_UINT16:
                return readIntegerJson<uint16_t>(stream);
            case BIN_KV_SERIALIZE_TYPE_UINT8:
                return readIntegerJson<uint8_t>(stream);
            case BIN_KV_SERIALIZE_TYPE_DOUBLE:
                return readPodJson<double>(stream);
            case BIN_KV_SERIALIZE_TYPE_BOOL:
                return JsonValue(read<uint8_t>(stream) != 0);
            case BIN_KV_SERIALIZE_TYPE_STRING:
                return readStringJson(stream);
            case BIN_KV_SERIALIZE_TYPE_OBJECT:
                return loadSection(stream);
            case BIN_KV_SERIALIZE_TYPE_ARRAY:
                return loadArray(stream, type);
            default:
                throw std::runtime_error("Unknown data type");
                break;
        }
    }

So this defines what all data types there are, and how they can be loaded. There are only a few options here. readIntegerJson() for different size of numbers (8 bit, 16 bit, ...), with sign or not. readPodJson() is where most of it seems to end up at with different types given

    template<typename T, typename JsonT = T>
    JsonValue readPodJson(Common::IInputStream &s)
    {
        JsonValue jv;
        jv = static_cast<JsonT>(readPod<T>(s));
        return jv;
    }

This just seems to read a number of bytes matching the given datatype from the stream, and treating this as the value of given type.

This is how the data seems to be formatted overall. First some size, then the data, ...











