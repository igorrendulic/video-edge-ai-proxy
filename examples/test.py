import os
import pathlib

folder = "/home/igor/Downloads/chrysalisedge/test"
video_fragments = os.listdir(folder)

for file in video_fragments:
    filename_timestamp = int(file.split("_")[0])
    file_duration = int(file.split("_")[1].split(".")[0])

    filepath = os.path.join(folder, file)
    fp = pathlib.Path(filepath)
    created_timestamp = int(fp.stat().st_mtime * 1000)

    print("timestamp: ", filename_timestamp, "\t created at: ", created_timestamp, "\t diff: ", created_timestamp - filename_timestamp, "\t duration: ", file_duration, "\t real diff:", (created_timestamp-filename_timestamp - file_duration) )