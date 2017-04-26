import json
import datetime
import subprocess


def monitorContainerNetworkSpeeds():
	freq = 1
	monitored = []

	lastRxBytes = {}
	lastTxBytes = {}

	docker_id = subprocess.check_output(['docker', 'ps', '-q']).split('\n')[:-1]
	for cid in docker_id:
		jsonObj = json.loads(subprocess.check_output(['docker', 'inspect', cid]).replace('\n', ' '))[0]
		pid = jsonObj['State']['Pid']
		cname = str(jsonObj['Name'][1:])
		if len(monitored) > 0 and cname not in monitored:
			continue
		print('Docker container: ' + cname)

		filename = '/proc/' + str(pid) + '/net/dev'
		try:
			devfile = open(filename, 'r')
			for line in devfile:
				s = line.split()
				if s[0] == 'eth0:':
					rxBytes = float(s[1])
					rxPackets = int(s[2])
					txBytes = float(s[9])
					txPackets = int(s[10])
					if cid not in lastRxBytes:
						lastRxBytes[cid] = rxBytes
					if cid not in lastTxBytes:
						lastTxBytes[cid] = txBytes
					print('Received: ' + str(rxBytes) + ' bytes ' + str(rxPackets) + ' packets')
					print('Sent: ' + str(txBytes) + ' bytes ' + str(txPackets) + ' packets')
					downlink = (rxBytes - lastRxBytes[cid]) / freq
					uplink = (txBytes - lastTxBytes[cid]) / freq
					if downlink < 1024:
						print('Downlink bandwidth: ' + str(downlink) + ' bytes/s')
					elif downlink < 1024 * 1024:
						print('Downlink bandwidth: ' + str(downlink / 1024) + ' kbytes/s')
					else:
						print('Downlink bandwidth: ' + str(downlink / 1024 / 1024) + ' mbytes/s')
					if uplink < 1024:
						print('Uplink bandwidth: ' + str(uplink) + ' bytes/s')
					elif uplink < 1024 * 1024:
						print('Uplink bandwidth: ' + str(uplink / 1024) + ' kbytes/s')
					else:
						print('Uplink bandwidth: ' + str(uplink / 1024 / 1024) + ' mbytes/s')

					if 'es' in locals():
						es.index(index = 'network-' + str(datetime.datetime.now().date()), doc_type = 'throughput', body = {'Pid': pid, 'Cid': cid, 'Name': cname, 'RxBytes': downlink, 'TxBytes': uplink, 'timestamp': datetime.datetime.utcnow()})
					lastRxBytes[cid] = rxBytes
					lastTxBytes[cid] = txBytes
					print('')
			devfile.close()
		except:
			continue

	return {'lastRxBytes':lastRxBytes, 'lastTxBytes':lastTxBytes}
