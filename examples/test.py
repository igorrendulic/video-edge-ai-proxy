# import os
# import pathlib

# folder = "/home/igor/Downloads/chrysalisedge/test"
# video_fragments = os.listdir(folder)

# for file in video_fragments:
#     filename_timestamp = int(file.split("_")[0])
#     file_duration = int(file.split("_")[1].split(".")[0])

#     filepath = os.path.join(folder, file)
#     fp = pathlib.Path(filepath)
#     created_timestamp = int(fp.stat().st_mtime * 1000)

#     print("timestamp: ", filename_timestamp, "\t created at: ", created_timestamp, "\t diff: ", created_timestamp - filename_timestamp, "\t duration: ", file_duration, "\t real diff:", (created_timestamp-filename_timestamp - file_duration) )


# https://stackoverflow.com/questions/53810984/mismatch-between-file-creation-time-and-current-time-in-python
# https://bugs.python.org/issue35522
# https://www.unix.com/tips-and-tutorials/20526-mtime-ctime-atime.html

import os, time

before_creation = time.time()

with open('my_file', 'w') as f:
    f.write('something')

creation_time = os.stat('my_file').st_ctime

print(before_creation)  # 1545031532.8819697
print(creation_time)    # 1545031532.8798425
print(before_creation < creation_time)  # False
print(before_creation - creation_time)