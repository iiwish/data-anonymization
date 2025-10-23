#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
数据匿名化与解密服务 - API测试脚本
用于手动测试接口，打印token和详细日志信息
"""

import json
import time
import hashlib
import hmac
import requests
import sys
from typing import Dict, Any, Optional

class DataAnonymizationTester:
    def __init__(self, base_url: str = "http://localhost:8080"):
        """
        初始化测试器
        
        Args:
            base_url: 服务基础URL，默认 http://localhost:8080
        """
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        
        # 测试用的系统配置（基于config.example.json）
        self.test_systems = {
            "BI_REPORT_SYSTEM": {
                "system_id": "BI_REPORT_SYSTEM",
                "shared_secret": "a_very_strong_and_long_secret_for_bi",
                "description": "商业智能报表分析系统"
            },
            "CUSTOMER_SERVICE_BOT": {
                "system_id": "CUSTOMER_SERVICE_BOT", 
                "shared_secret": "another_unique_secret_for_the_chatbot",
                "description": "客户服务聊天机器人"
            }
        }
    
    def generate_signature(self, system_id: str, user_id: str, request_body: str) -> Dict[str, str]:
        """
        生成HMAC签名
        
        Args:
            system_id: 系统ID
            user_id: 用户ID
            request_body: 请求体JSON字符串
            
        Returns:
            包含签名信息的字典
        """
        if system_id not in self.test_systems:
            raise ValueError(f"未知的系统ID: {system_id}")
            
        secret = self.test_systems[system_id]["shared_secret"]
        
        # 计算请求体的SHA256
        body_hash = hashlib.sha256(request_body.encode('utf-8')).hexdigest()
        
        # 生成时间戳
        timestamp = str(int(time.time()))
        
        # 构建签名内容
        sign_content = system_id + user_id + timestamp + body_hash
        
        # 计算HMAC-SHA256
        h = hmac.new(
            secret.encode('utf-8'),
            sign_content.encode('utf-8'),
            hashlib.sha256
        )
        signature = h.hexdigest()
        
        return {
            "system_id": system_id,
            "user_id": user_id,
            "timestamp": timestamp,
            "signature": signature,
            "body_hash": body_hash,
            "sign_content": sign_content
        }
    
    def build_auth_header(self, signature_info: Dict[str, str]) -> str:
        """
        构建Authorization头
        
        Args:
            signature_info: 签名信息字典
            
        Returns:
            Authorization头字符串
        """
        return (
            f"MCP-HMAC-SHA256 "
            f"SystemID={signature_info['system_id']},"
            f"UserID={signature_info['user_id']},"
            f"Timestamp={signature_info['timestamp']},"
            f"Signature={signature_info['signature']}"
        )
    
    def print_debug_info(self, signature_info: Dict[str, str], request_body: str, endpoint: str):
        """
        打印调试信息
        
        Args:
            signature_info: 签名信息
            request_body: 请求体
            endpoint: 接口端点
        """
        print("\n" + "="*80)
        print("🔍 调试信息")
        print("="*80)
        print(f"📡 目标接口: {self.base_url}{endpoint}")
        print(f"👤 系统ID: {signature_info['system_id']}")
        print(f"👤 用户ID: {signature_info['user_id']}")
        print(f"⏰ 时间戳: {signature_info['timestamp']}")
        print(f"🔑 签名: {signature_info['signature']}")
        print(f"📄 请求体哈希: {signature_info['body_hash']}")
        print(f"📝 签名内容: {signature_info['sign_content']}")
        print(f"🔐 Authorization头:")
        print(f"   {self.build_auth_header(signature_info)}")
        print(f"📦 请求体:")
        print(json.dumps(json.loads(request_body), indent=2, ensure_ascii=False))
        print("="*80 + "\n")
    
    def test_anonymize(self, system_id: str = "BI_REPORT_SYSTEM", user_id: str = "test_user_001"):
        """
        测试匿名化接口
        
        Args:
            system_id: 系统ID
            user_id: 用户ID
        """
        print("🚀 开始测试匿名化接口")
        
        # 构建请求体
        request_body = {
            "session_id": f"sess_{int(time.time())}",
            "payload": {
                "metadata": {
                    "report_name": "Q3 Sales Analysis for {华东}",
                    "requester": user_id
                },
                "analysis_prompt": "Analyze the following sales data. The previous quarter's top product was '手机'. Focus on the performance of '华东' and compare it with other regions. The total revenue for Q2 was 1500000.",
                "data_table": [
                    {
                        "区域": "华东",
                        "核心产品": "手机", 
                        "季度收入": 1500000,
                        "同比增长率": "12.5%",
                        "活跃用户数": 12000
                    },
                    {
                        "区域": "华北",
                        "核心产品": "电脑",
                        "季度收入": 950000,
                        "同比增长率": "-3.2%", 
                        "活跃用户数": 8500
                    }
                ]
            },
            "anonymization_rules": [
                {
                    "strategy": "MAP_CODE",
                    "applies_to": {"type": "REGION", "values": ["华东", "华北"]}
                },
                {
                    "strategy": "MAP_CODE", 
                    "applies_to": {"type": "PRODUCT", "values": ["手机", "电脑"]}
                },
                {
                    "strategy": "TRANSFORM",
                    "strategy_params": {"noise_level": 0.05},
                    "applies_to": {"type": "REVENUE", "values": [1500000, 950000]}
                },
                {
                    "strategy": "MAP_PLACEHOLDER",
                    "applies_to": {"type": "USER_COUNT", "values": [12000, 8500]}
                },
                {
                    "strategy": "PASSTHROUGH", 
                    "applies_to": {"type": "GROWTH_RATE", "values": ["12.5%", "-3.2%"]}
                }
            ]
        }
        
        request_body_str = json.dumps(request_body, ensure_ascii=False)
        signature_info = self.generate_signature(system_id, user_id, request_body_str)
        
        # 打印调试信息
        self.print_debug_info(signature_info, request_body_str, "/v1/anonymize")
        
        # 发送请求
        headers = {
            "Authorization": self.build_auth_header(signature_info),
            "Content-Type": "application/json"
        }
        
        try:
            response = self.session.post(
                f"{self.base_url}/v1/anonymize",
                headers=headers,
                data=request_body_str.encode('utf-8'),
                timeout=30
            )
            
            print("📡 响应信息:")
            print(f"   状态码: {response.status_code}")
            print(f"   响应头: {dict(response.headers)}")
            
            if response.status_code == 200:
                result = response.json()
                print("✅ 匿名化成功!")
                print(json.dumps(result, indent=2, ensure_ascii=False))
                
                # 保存映射表供解密测试使用
                if "mappings_to_store" in result:
                    with open("test_mappings.json", "w", encoding="utf-8") as f:
                        json.dump(result["mappings_to_store"], f, indent=2, ensure_ascii=False)
                    print(f"💾 映射表已保存到: test_mappings.json")
                    
                return result
            else:
                print(f"❌ 请求失败: {response.status_code}")
                print(f"   错误信息: {response.text}")
                return None
                
        except requests.exceptions.RequestException as e:
            print(f"❌ 请求异常: {e}")
            return None
    
    def test_decrypt_json(self, system_id: str = "BI_REPORT_SYSTEM", user_id: str = "test_user_001"):
        """
        测试解密接口（JSON格式）
        
        Args:
            system_id: 系统ID
            user_id: 用户ID
        """
        print("🚀 开始测试解密接口（JSON格式）")
        
        # 尝试加载之前保存的映射表
        # try:
        #     with open("test_mappings.json", "r", encoding="utf-8") as f:
        #         mappings = json.load(f)
        # except FileNotFoundError:
        #     print("⚠️  未找到映射表文件，使用示例映射")
        mappings = {
            "categorical_mappings": {
                "REGION": {"REGION_a3f5": "华东", "REGION_b1e9": "华北"},
                "PRODUCT": {"PRODUCT_c8b1": "手机", "PRODUCT_d2a7": "电脑"}
            },
            "metric_placeholder_mappings": {
                "USER_COUNT_plc_1": 12000,
                "USER_COUNT_plc_2": 8500
            }
        }
        
        # 构建请求体（JSON格式）
        request_body = {
            "data_with_anonymized_codes": {
                "summary": "REGION_a3f5 区域表现突出，主要贡献来自 PRODUCT_c8b1。",
                "key_findings": [
                    {"dimension": "区域", "value": "REGION_a3f5"},
                    {"dimension": "产品", "value": "PRODUCT_c8b1"}
                ]
            },
            "mappings": mappings
        }
        
        request_body_str = json.dumps(request_body, ensure_ascii=False)
        signature_info = self.generate_signature(system_id, user_id, request_body_str)
        
        # 打印调试信息
        self.print_debug_info(signature_info, request_body_str, "/v1/decrypt")
        
        # 发送请求
        headers = {
            "Authorization": self.build_auth_header(signature_info),
            "Content-Type": "application/json"
        }
        
        try:
            response = self.session.post(
                f"{self.base_url}/v1/decrypt",
                headers=headers,
                data=request_body_str.encode('utf-8'),
                timeout=30
            )
            
            print("📡 响应信息:")
            print(f"   状态码: {response.status_code}")
            print(f"   响应头: {dict(response.headers)}")
            
            if response.status_code == 200:
                result = response.json()
                print("✅ 解密成功!")
                print(json.dumps(result, indent=2, ensure_ascii=False))
                return result
            else:
                print(f"❌ 请求失败: {response.status_code}")
                print(f"   错误信息: {response.text}")
                return None
                
        except requests.exceptions.RequestException as e:
            print(f"❌ 请求异常: {e}")
            return None
    
    def test_decrypt_text(self, system_id: str = "BI_REPORT_SYSTEM", user_id: str = "test_user_001"):
        """
        测试解密接口（纯文本格式）
        
        Args:
            system_id: 系统ID
            user_id: 用户ID
        """
        print("🚀 开始测试解密接口（纯文本格式）")
        
        # 尝试加载之前保存的映射表
        try:
            with open("test_mappings.json", "r", encoding="utf-8") as f:
                mappings = json.load(f)
        except FileNotFoundError:
            print("⚠️  未找到映射表文件，使用示例映射")
            mappings = {
                "categorical_mappings": {
                    "REGION": {"REGION_a3f5": "华东", "REGION_b1e9": "华北"},
                    "PRODUCT": {"PRODUCT_c8b1": "手机", "PRODUCT_d2a7": "电脑"}
                },
                "metric_placeholder_mappings": {
                    "USER_COUNT_plc_1": 12000,
                    "USER_COUNT_plc_2": 8500
                }
            }
        
        # 构建请求体（纯文本格式）
        request_body = {
            "data_with_anonymized_codes": "分析显示，REGION_a3f5 区域的 PRODUCT_c8b1 表现最佳，活跃用户数为 USER_COUNT_plc_1。",
            "mappings": mappings
        }
        
        request_body_str = json.dumps(request_body, ensure_ascii=False)
        signature_info = self.generate_signature(system_id, user_id, request_body_str)
        
        # 打印调试信息
        self.print_debug_info(signature_info, request_body_str, "/v1/decrypt")
        
        # 发送请求
        headers = {
            "Authorization": self.build_auth_header(signature_info),
            "Content-Type": "application/json"
        }
        
        try:
            response = self.session.post(
                f"{self.base_url}/v1/decrypt",
                headers=headers,
                data=request_body_str.encode('utf-8'),
                timeout=30
            )
            
            print("📡 响应信息:")
            print(f"   状态码: {response.status_code}")
            print(f"   响应头: {dict(response.headers)}")
            
            if response.status_code == 200:
                result = response.json()
                print("✅ 解密成功!")
                print(json.dumps(result, indent=2, ensure_ascii=False))
                return result
            else:
                print(f"❌ 请求失败: {response.status_code}")
                print(f"   错误信息: {response.text}")
                return None
                
        except requests.exceptions.RequestException as e:
            print(f"❌ 请求异常: {e}")
            return None
    
    def run_all_tests(self):
        """运行所有测试"""
        print("🎯 开始执行完整测试流程")
        print(f"🔗 目标服务: {self.base_url}")
        print()
        
        # 1. 测试匿名化接口
        anonymize_result = self.test_anonymize()
        if not anonymize_result:
            print("❌ 匿名化测试失败，跳过后续测试")
            return
        
        print("\n" + "="*80)
        print("⏳ 等待2秒后继续解密测试...")
        print("="*80)
        time.sleep(2)
        
        # 2. 测试解密接口（JSON格式）
        decrypt_json_result = self.test_decrypt_json()
        
        print("\n" + "="*80)
        print("⏳ 等待2秒后继续文本解密测试...")
        print("="*80)
        time.sleep(2)
        
        # 3. 测试解密接口（纯文本格式）
        decrypt_text_result = self.test_decrypt_text()
        
        print("\n" + "="*80)
        print("📊 测试总结")
        print("="*80)
        print(f"✅ 匿名化测试: {'成功' if anonymize_result else '失败'}")
        print(f"✅ JSON解密测试: {'成功' if decrypt_json_result else '失败'}")
        print(f"✅ 文本解密测试: {'成功' if decrypt_text_result else '失败'}")
        print("="*80)


def main():
    """主函数"""
    # 解析命令行参数
    if len(sys.argv) > 1:
        base_url = sys.argv[1]
    else:
        base_url = "http://localhost:8080"
    
    tester = DataAnonymizationTester(base_url)
    
    print("🔧 数据匿名化与解密服务 - API测试工具")
    print("="*50)
    
    while True:
        print("\n请选择测试选项:")
        print("1. 运行完整测试流程")
        print("2. 仅测试匿名化接口")
        print("3. 仅测试解密接口（JSON格式）")
        print("4. 仅测试解密接口（纯文本格式）")
        print("5. 退出")
        
        choice = input("\n请输入选项 (1-5): ").strip()
        
        if choice == "1":
            tester.run_all_tests()
        elif choice == "2":
            tester.test_anonymize()
        elif choice == "3":
            tester.test_decrypt_json()
        elif choice == "4":
            tester.test_decrypt_text()
        elif choice == "5":
            print("👋 退出测试工具")
            break
        else:
            print("❌ 无效选项，请重新输入")


if __name__ == "__main__":
    main()