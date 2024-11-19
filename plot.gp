set terminal pngcairo size 1024,768
set output "MFCC_Spectrogram.png"
set xlabel "Time (frames)"
set ylabel "MFCC Coefficients"
set title "MFCC Spectrogram"
set palette defined (0 "blue", 1 "cyan", 2 "green", 3 "yellow", 4 "red")
unset key
plot "MFCC_data.txt" matrix with image