#include<iostream>
#include<string>
#include<fstream>
#include<vector>
#include<thread>
#include<chrono>
#include"json.hpp"
static nlohmann::json total = nlohmann::json().array();
std::string token;
std::vector<std::string> city = {"Taipei","NewTaipei","Taoyuan","Taichung","Tainan","Kaohsiung","Keelung","Hsinchu", "HsinchuCounty","MiaoliCounty","ChanghuaCounty","NantouCounty","YunlinCounty","ChiayiCounty","Chiayi","PingtungCounty","YilanCounty","HualienCounty","TaitungCounty","KinmenCounty","PenghuCounty","LienchiangCounty"};
void gettoken(const char *id,const char *secret) {
    if (!id || !secret) {
        std::cerr << "test" << '\n';
        return;
    }
    std::string s = "curl --request POST https://tdx.transportdata.tw/auth/realms/TDXConnect/protocol/openid-connect/token "
                    "--header 'content-type: application/x-www-form-urlencoded' "
                    "--data 'grant_type=client_credentials&client_id=" + std::string(id) +
                    "&client_secret="+ std::string(secret) + "' > token.json";
    system(s.c_str());
    std::ifstream f("token.json");
    nlohmann::json j = nlohmann::json::parse(f);
    if (j.contains("access_token")) token = j["access_token"];
    f.close();
    remove("token.json");
    std::cerr << token << '\n';
}
void getcity() {
    for (int i = 0;i < 22;i++) {
        std::string s = "curl -X 'GET' "
                        "-H 'authorization: Bearer " + token + "' " +
                        "-H 'Content-Encoding: br,gzip' " +
                        "-H 'Content-Type: application/json' " +
                        "\'https://tdx.transportdata.tw/api/basic/v2/Bus/Route/City/" +city[i] +"?$select=RouteUID,RouteID,RouteName,BusRouteType,DepartureStopNameZh,DestinationStopNameZh&$format=JSON\' > temp.json";
        system(s.c_str());
        std::ifstream f("temp.json");
        if (f.good()) {
            nlohmann::json json = nlohmann::json::parse(f);
            if (json.is_array()) {
                for (auto &j : json) {
                    nlohmann::json temp;
                    temp["RouteUID"] = j["RouteUID"];
                    temp["RouteID"] = j["RouteID"];
                    temp["RouteName"] = j["RouteName"]["Zh_tw"];
                    temp["City"] = city[i];
                    temp["Type"] = j["BusRouteType"];
                    temp["DepartureStopNameZh"] = j["DepartureStopNameZh"];
                    temp["DestinationStopNameZh"] = j["DestinationStopNameZh"];
                    total.emplace_back(temp);
                }
            }
            else std::cout << "rate\n";
        }
        f.close();
        if ((i+1) % 5 == 0) {
            std::cout << "sleep\n";
            std::this_thread::sleep_for(std::chrono::seconds(60));
        }
    }
    remove("temp.json");
}
void getinter() {
    std::string s = "curl -X 'GET' "
                    "-H 'authorization: Bearer " + token + "' " +
                    "-H 'Content-Encoding: br,gzip' " +
                    "-H 'Content-Type: application/json' " +
                    "\'https://tdx.transportdata.tw/api/basic/v2/Bus/Route/InterCity?$select=RouteUID,RouteID,RouteName,BusRouteType,DepartureStopNameZh,DestinationStopNameZh&$format=JSON\' > temp.json";
    system(s.c_str());
    std::ifstream f("temp.json");
    if (f.good()) {
        if (f.good()) {
            nlohmann::json json = nlohmann::json::parse(f);
            for (auto &j : json) {
                nlohmann::json temp;
                temp["RouteUID"] = j["RouteUID"];
                temp["RouteName"] = j["RouteName"]["Zh_tw"];
                temp["RouteID"] = j["RouteID"];
                temp["City"] = "InterCity";
                temp["Type"] = j["BusRouteType"];
                temp["DepartureStopNameZh"] = j["DepartureStopNameZh"];
                temp["DestinationStopNameZh"] = j["DestinationStopNameZh"];
                total.emplace_back(temp);
            }
        }
    }
    f.close();
    remove("temp.json");
}
int main() {
    const char *id = getenv("Client_ID"),*secret = getenv("Client_SECRET");
    try {
        gettoken(id, secret);
        getcity();
        getinter();
        std::ofstream out("routes.json");
        out << total.dump();
        out.close();
        std::cout << "test " << total.size() << " \n";
    }
    catch (std::exception &e) {
        std::cerr << "error: " << e.what() << '\n';
    }
    return 0;
}