from rtlsdr import RtlSdr
import pylab
from matplotlib.mlab import psd


# lifted mostly from https://github.com/roger-/pyrtlsdr/blob/master/demo_waterfall.py

NFFT = 1024
NUM_SAMPLES_PER_SCAN = NFFT*16
NUM_BUFFERED_SWEEPS = 100
NUM_SCANS_PER_SWEEP = 1


sdr = RtlSdr()

sdr.rs = 2.4e6
sdr.fc = 100e6
sdr.gain = 'auto'

fc = sdr.fc
rs = sdr.rs
freq_range = (fc - rs/2)/1e6, (fc + rs*(NUM_SCANS_PER_SWEEP - 0.5))/1e6

start = 24e6
stop = 24.05e6
#stop = 1766e6
print int(start)
print int(stop)
fcs = range(int(start), int(stop), 500)
print len(fcs)

D = []

#for scan_nu#m, start_ind in enumerate(range(0, NUM_SCANS_PER_SWEEP*NFFT, NFFT)):
for fc in fcs:
	sdr.fc = fc
	print sdr.fc

	# estimate PSD for one scan
	samples = sdr.read_samples(NUM_SAMPLES_PER_SCAN)
	psd_scan, f = psd(samples, NFFT=NFFT)

	d = psd_scan[NFFT/2:NFFT/2+NFFT/4]
	D.extend(d)
#	df = f[NFFT/2:NFFT/2+NFFT/4]

pylab.plot(D)
pylab.show()

sdr.close()
