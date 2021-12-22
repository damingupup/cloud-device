# encoding: utf-8
import subprocess
import sys
import uuid
import os
import shutil
import zipfile

script_path = os.path.dirname(__file__)  # 脚本所在目录
sign_extensions = ['.framework/', '.dylib', '.appex/', '.app/']


class ResginUtils(object):

    @staticmethod
    def execute_cmd(cmd):
        process = os.popen(cmd)
        output = process.read()
        process.close()
        return output

    @staticmethod
    def copy(file, dst):
        shutil.copy(file, dst)

    @staticmethod
    def rename(file_path, new_file_name):
        fileDir, fileName = os.path.split(file_path)
        os.rename(file_path, os.path.join(fileDir, new_file_name))

    # 删除目录
    @staticmethod
    def remove_dir(path):
        if os.path.exists(path):
            shutil.rmtree(path)

    # 解压文件
    @staticmethod
    def unzip_file(source_file, output_path):
        zip_obj = zipfile.ZipFile(source_file, 'r')
        zip_file_list = zip_obj.namelist()
        zip_file_list.reverse()
        zip_obj.extractall(output_path)
        zip_obj.close()
        return zip_file_list

    # 压缩文件
    @staticmethod
    def zip_file(source_path, output_path):
        zip_file = zipfile.ZipFile(output_path, 'w')
        pre_len = len(os.path.dirname(source_path))
        for parent, dir_names, file_names in os.walk(source_path):
            for file_name in file_names:
                pathFile = os.path.join(parent, file_name)
                arc_name = pathFile[pre_len:].strip(os.path.sep)
                zip_file.write(pathFile, arc_name)
        zip_file.close()

    # 拼接空白路径
    @staticmethod
    def handleWhiteSpace(name):
        space = name.lstrip().rstrip()
        return space.replace('\n', '')

    @staticmethod
    def is_need_sign(file_name):
        for sign_extension in sign_extensions:
            if sign_extension == file_name[file_name.rfind('.'):]:
                return True
        return False


resign_utils = ResginUtils()


# 重签名流程
# 1.解压ipa
# 2.删除旧签名
# 3.复制新描述文件
# 4.用新的证书
# 5.压缩ipa
# security find-identity -v -p codesigning 可以查看当前电脑安装的证书

class Resign:
    def __init__(self, ipa_name,path):
        # ipa解压后文件列表
        self.zip_file_list = None
        # 解压后的app路径
#         self.app_temp_path = './temp'
        self.app_temp_path = path
        self.current_path = os.path.dirname(__file__)  # 脚本所在目录
        self.temp_file_dir = os.path.join(path, str(uuid.uuid4()))  # 临时文件夹目录
        os.makedirs(self.temp_file_dir)
        self.entitle_plist = os.path.join(self.temp_file_dir, 'entitlements.plist')  # 生成的plist信息
        self.output_ipa_path = os.path.join(path,'%s' % ipa_name)  # 默认输出ipa文件的路径
        print('output_ipa_path ==>%s' % self.output_ipa_path)

    # 处理原ipa文件
    def unzip_ipa(self, ipa_path):
        if os.path.isfile(ipa_path):
            if '.ipa' in ipa_path:
                resign_utils.remove_dir(self.temp_file_dir)
                self.zip_file_list = resign_utils.unzip_file(ipa_path, self.temp_file_dir)
                payload_path = os.path.join(self.temp_file_dir, 'Payload')
                app_PackageName = resign_utils.execute_cmd('ls %s' % (payload_path))
                app_path = os.path.join(payload_path, app_PackageName)
                self.app_temp_path = resign_utils.handleWhiteSpace(app_path)
                print('app_temp_path   ==>%s' % self.app_temp_path)
                self._check_code_sign(self.app_temp_path)
                return True
            else:
                print('文件格式非法')
                return False
        else:
            print('文件不存在')
            return False

    def zip_ipa(self):
        resign_utils.zip_file(os.path.join(self.temp_file_dir, 'Payload'), self.output_ipa_path)
        resign_utils.remove_dir(self.temp_file_dir)

    # 根据pp文件导出Entitlements信息
    def export_sign_info(self, pp_path):
        temp_plist = os.path.join(self.temp_file_dir, 'entitlements_temp.plist')

        cmd = 'security cms -D -i "%s" > %s' % (pp_path, temp_plist)
        print(cmd)
        result = resign_utils.execute_cmd(cmd)
        print(result)
        cmd = '/usr/libexec/PlistBuddy -x -c "Print:Entitlements" %s > %s' % (temp_plist, self.entitle_plist)
        result = resign_utils.execute_cmd(cmd)
        print(result)
        print(pp_path, self.app_temp_path)
        resign_utils.copy(pp_path, self.app_temp_path)
        file_dir, file_name = os.path.split(pp_path)
        resign_utils.rename(os.path.join(self.app_temp_path, file_name), 'embedded.mobileprovision')
        os.remove(temp_plist)

    def code_sign(self, cert_name, file_path):
        cmd = 'codesign -f -s "%s" --entitlements %s "%s"' % (cert_name, self.entitle_plist, file_path)
        print(cmd)
        resign_utils.execute_cmd(cmd)

    def sign_start(self, cer_name):
        for file_name in self.zip_file_list:
            if resign_utils.is_need_sign(file_name):
                self.code_sign(cer_name, os.path.join(self.temp_file_dir, file_name))
        os.remove(self.entitle_plist)
        self._check_code_sign(self.app_temp_path)

    @staticmethod
    def _check_code_sign(package_path):
        cmd = f'codesign -vv -d {package_path}'
        # result = resign_utils.execute_cmd(cmd)
        os.system(cmd)
        print("dddddddd")
        print("dddddddd")

if __name__ == '__main__':
    params = sys.argv
    ipa_path = "/Users/yuzhiming/go/project/ctp-ios-proxy/yoka_0_TEST_2021_07_13_18_17.ipa"
    ipa_name = ipa_path.split("/")[-1]
    path = os.path.dirname(ipa_path)
    print("开始重签名")
    print(ipa_name)
    print(ipa_path)
    resign = Resign(ipa_name,path)
    if resign.unzip_ipa(ipa_path):
        # 描述文件，如果需要查看系统中已经保存的描述文件：~/Library/MobileDevice/Provisioning\ Profiles/
        home = os.environ['HOME']
        pp_dir = os.path.join(home, "Library/MobileDevice/Provisioning Profiles/")
        pp_path_list = os.listdir(pp_dir)
        pp_path = ""
        for i in pp_path_list:
            if "mobileprovision" in i:
                pp_path = os.path.join(pp_dir, i)
                break
        # print(pp_path)
        print("-----------------------")
        pp_path = "/Users/yuzhiming/go/project/ctp-ios-proxy/Resign.mobileprovision"
        # 需要本机安装 相应证书，查看安装的证书使用命令：security find-identity -p codesigning -v
        cmd = "security find-identity -p codesigning -v"
        data = subprocess.Popen(cmd, stdin=subprocess.PIPE, stderr=subprocess.PIPE,
                                stdout=subprocess.PIPE, universal_newlines=True, shell=True)
        cer_name = ""
        for i in data.stdout.readlines():
            if "iPhone Developer" in i:
                cer_name = i.split(" ")[3]
                break
        resign.export_sign_info(pp_path)
        resign.sign_start(cer_name)
        resign.zip_ipa()
