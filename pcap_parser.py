__author__ = 'LeTurt'

#pip3 install dpkt to run first
#then run with pcap file path as parameter
#result should be directory "levin_pcap_dumps", under which a sub-directory for each command id found in the dump.
#under which two files for each packet with that command, one as text-formatted hex-string, one as pcap file for opening with wireshark

import dpkt
import sys
import os

TURTLE_PORT = 11897 #i just assume the daemon uses this port, modify if needed. i got it from turtlecoin source and from netstat
LEVIN_SIG = "0121010101010101" #the signature for levin protocol header packets, from turtlecoin (cryptonote) code as well

print('Argument List:', str(sys.argv))
f = open(sys.argv[1], "rb")
pcap = dpkt.pcap.Reader(f)
index = 1
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
	#header is encoded in little-endian, so have to reverse the bytes for command first
	byte1_hex = data_str[40:42]
	byte2_hex = data_str[38:40]
	byte3_hex = data_str[36:38]
	byte4_hex = data_str[34:36]
	cmd_hex = byte1_hex+byte2_hex+byte3_hex+byte4_hex
#	print("hex", cmd_hex)
	cmd_id = int(cmd_hex, 16) #this should not be the levin protocol command id for msg, as integer
	dump_dir = "levin_pcap_dumps/"+str(cmd_id)
	print("cmd:", cmd_id)
	if not os.path.exists(dump_dir):
		os.makedirs(dump_dir)

	#first dump hex-string
	hex_file = open(dump_dir+"/"+str(index)+".txt", "w")
	hex_file.write(data_str)
	hex_file.close()

	#dump packet pcap for wireshark analysis
	pcap_file = open(dump_dir+"/"+str(index)+".pcap", "wb")
#	f = open(pcap_file, "rb")
	pcap_writer = dpkt.pcap.Writer(pcap_file)
	pcap_writer.writepkt(buf, ts)
#	pcap_file.write(buf)
	pcap_file.close()

	index += 1
#	print(tcp.data.hex())

