#include<iostream>
#include<string>
#include<fstream>
#include<vector>
#include<chrono>
#include<set>
#include<thread>
#include"nlohmann/json.hpp"
static nlohmann::json total = nlohmann::json::array();
static std::vector<std::string> city = {"Taipei","NewTaipei","Taoyuan","Taichung","Tainan","Kaohsiung","Keelung","Hsinchu", "HsinchuCounty","MiaoliCounty","ChanghuaCounty","NantouCounty","YunlinCounty","ChiayiCounty","Chiayi","PingtungCounty","YilanCounty","HualienCounty","TaitungCounty","KinmenCounty","PenghuCounty","LienchiangCounty"};
std::string token;
void gettoken(const char *id,const char *secret) {
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
}
void getcity() {
    for (int i = 0;i < 22;i++) {
        std::string s = "curl -X 'GET' "
                        "-H 'authorization: Bearer " + token + "' " +
                        "-H 'Content-Encoding: br,gzip' " +
                        "-H 'Content-Type: application/json' " +
                        "\'https://tdx.transportdata.tw/api/basic/v2/Bus/Route/City/" +city[i] +"?$select=HasSubRoutes,RouteUID,RouteID,RouteName,BusRouteType,DepartureStopNameZh,DestinationStopNameZh,SubRoutes&$format=JSON\' > temp.json";
        system(s.c_str());
        std::ifstream f("temp.json");
        if (f.good()) {
            nlohmann::json json = nlohmann::json::parse(f);
            if (json.is_array()) {
                for (auto &j : json) {
                    if (j["HasSubRoutes"] == true) {
                        std::set<std::string> set;
                        for (auto &k : j["SubRoutes"]) {
                            if(set.find(k["SubRouteUID"].get<std::string>()) != set.end()) continue;
                            set.emplace(k["SubRouteUID"].get<std::string>());
                            nlohmann::json temp;
                            temp["RouteUID"] = j["RouteUID"];
                            temp["RouteName"] = j["RouteName"]["Zh_tw"];
                            temp["City"] = city[i];
                            temp["Type"] = j["BusRouteType"];
                            temp["SubRouteUID"] = k["SubRouteUID"];
                            temp["SubRouteName"] = k["SubRouteName"]["Zh_tw"];
                            temp["DestinationStopNameZh"] = k.contains("DestinationStopNameZh") ? k["DestinationStopNameZh"] : j["DestinationStopNameZh"];
                            temp["DepartureStopNameZh"] = k.contains("DepartureStopNameZh") ? k["DepartureStopNameZh"] : j["DepartureStopNameZh"];
                            total.emplace_back(temp);
                        }
                    }
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
                    "\'https://tdx.transportdata.tw/api/basic/v2/Bus/Route/InterCity?$select=HasSubRoutes,RouteUID,RouteID,RouteName,BusRouteType,DepartureStopNameZh,DestinationStopNameZh,SubRoutes&$format=JSON\' > temp.json";
    system(s.c_str());
    std::ifstream f("temp.json");
    if (f.good()) {
        nlohmann::json json = nlohmann::json::parse(f);
        if (json.is_array()) {
            for (auto &j : json) {
                if (j["HasSubRoutes"] == true) {
                    std::set<std::string> set;
                    for (auto &k : j["SubRoutes"]) {
                        if(set.find(k["SubRouteUID"].get<std::string>()) != set.end()) continue;
                        set.emplace(k["SubRouteUID"].get<std::string>());
                        nlohmann::json temp;
                        temp["RouteUID"] = j["RouteUID"];
                        temp["RouteName"] = j["RouteName"]["Zh_tw"];
                        temp["City"] = "InterCity";
                        temp["Type"] = j["BusRouteType"];
                        temp["SubRouteUID"] = k["SubRouteUID"];
                        temp["SubRouteName"] = k["SubRouteName"]["Zh_tw"];
                        temp["DestinationStopNameZh"] = k.contains("DestinationStopNameZh") ? k["DestinationStopNameZh"] : j["DestinationStopNameZh"];
                        temp["DepartureStopNameZh"] = k.contains("DepartureStopNameZh") ? k["DepartureStopNameZh"] : j["DepartureStopNameZh"];
                        total.emplace_back(temp);
                    }
                }
            }
        }
        else std::cout << "rate\n";
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
    }
    catch (std::exception &e) {
        std::cerr << "error: " << e.what() << '\n';
    }
}