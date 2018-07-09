__author__ = 'LeTurt'

#pip3 install dpkt to run first
#then run with pcap file path as parameter
#result should be directory "levin_pcap_dumps", under which a sub-directory for each command id found in the dump.
#under which three files for each packet with that command, one as text-formatted hex-string, one as pcap file for opening with wireshark,
#one as raw binary of the tcp payload, which would be the actual protocol packet and nothing more
#there is also the "all" directory, under which a summary txt file is placed containing hex strings for all packets, one on line. easier comparisons.

import dpkt
import sys
import os

TURTLE_PORT = 11897 #i just assume the daemon uses this port, modify if needed. i got it from turtlecoin source and from netstat
LEVIN_SIG = "0121010101010101" #the signature for levin protocol header packets, from turtlecoin (cryptonote) code as well

print('Argument List:', str(sys.argv))
f = open(sys.argv[1], "rb")
pcap = dpkt.pcap.Reader(f)
index = 1
all_hex = ""
for ts, buf in pcap:
#	print(ts, len(buf))
	hex = buf.hex()
	eth = dpkt.ethernet.Ethernet(buf)
	ip = eth.data
	tcp = ip.data
#	print("from ", tcp.sport, " to ", tcp.dport, "len ", len(tcp.data))
	if tcp.sport != TURTLE_PORT and tcp.dport != TURTLE_PORT:
		continue
	if len(tcp.data) == 0:
		continue
	data_str = tcp.data.hex()
	start = data_str[0:16]
	if start != LEVIN_SIG:
		#large data streams get chunked, so this assumes the same msg spans multiple IP packets and skips parsing ones without header id
		print("skipping non-levin header packet")
		continue
	#need to get command id, which is at index 34-42 in hex string (17-21 in byte array but string has 2 chars for byte)
	#header is encoded in little-endian, so have to reverse the bytes for command first
	byte1_hex = data_str[40:42]
	byte2_hex = data_str[38:40]
	byte3_hex = data_str[36:38]
	byte4_hex = data_str[34:36]
	cmd_hex = byte1_hex+byte2_hex+byte3_hex+byte4_hex
#	print("hex", cmd_hex)
	cmd_id = int(cmd_hex, 16) #this should now be the levin protocol command id for msg, as integer
	dump_dir = "levin_pcap_dumps/"+str(cmd_id)
	print("cmd:", cmd_id)
	if not os.path.exists(dump_dir):
		os.makedirs(dump_dir)

	#first dump hex-string
	hex_file = open(dump_dir+"/"+str(index)+".txt", "w")
	hex_file.write(data_str)
	hex_file.close()

	all_hex += str(cmd_id)+": "+data_str+"\n"

	#dump packet pcap for wireshark analysis
	pcap_file = open(dump_dir+"/"+str(index)+".pcap", "wb")
	pcap_writer = dpkt.pcap.Writer(pcap_file)
	pcap_writer.writepkt(buf, ts)
	pcap_file.close()

	#dump packet binary for parser testing
	bin_file = open(dump_dir+"/"+str(index)+".bin", "wb")
	bin_file.write(tcp.data)
	bin_file.close()

	index += 1
#	print(tcp.data.hex())

dump_dir = "levin_pcap_dumps/all/"
if not os.path.exists(dump_dir):
	os.makedirs(dump_dir)
hex_file = open(dump_dir + "all_cmds_hex.txt", "w")
hex_file.write(all_hex)
hex_file.close()

