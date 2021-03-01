# Copyright 2020 Wearless Tech Inc All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import time
import os
import threading
import sched
from datetime import timedelta, datetime
import re

class CleanupScheduler(threading.Thread):

    def __init__(self, folder, device, remove_older_than):
        threading.Thread.__init__(self)
        self.__folder = folder
        self.__device = device
        self.__remove_older_than = remove_older_than
        self.__scheduler = sched.scheduler(time.time, time.sleep)
        self.__units = {'s':'seconds', 'm':'minutes', 'h':'hours', 'd':'days', 'w':'weeks'}

        self.__delay_seconds = self.convert_to_seconds(remove_older_than)

    def convert_to_seconds(self, s):
        return int(timedelta(**{
            self.__units.get(m.group('unit').lower(), 'seconds'): int(m.group('val'))
            for m in re.finditer(r'(?P<val>\d+)(?P<unit>[smhdw]?)', s, flags=re.I)
        }).total_seconds())

    def remove_mp4_files(self):
        try:
            now = int(time.time() * 1000)
            remove_older_than = now - (self.__delay_seconds * 1000)

            # print("removing older mp4 files", self.__folder + "/" + self.__device, datetime.utcfromtimestamp(remove_older_than/1000).strftime('%Y-%m-%d %H:%M:%S'))
            files = os.listdir(self.__folder + "/" + self.__device)

            if len(files) > 0:
                for f in files:
                    splitted =  f.split("_")
                    if len(splitted) == 2:
                        file_timestamp = int(splitted[0])
                        if file_timestamp < remove_older_than:
                            rm_filename = self.__folder + "/" + self.__device + "/" + f
                            if os.path.exists(rm_filename):
                                os.remove(rm_filename)
                            
            # print("numer of listed files: ", str(len(files)))

        except Exception as e:
            print("failed delete files", e)
    
    def run(self):
        while True:
            self.__scheduler.enter(5, 1, self.remove_mp4_files)
            self.__scheduler.run()

    

    