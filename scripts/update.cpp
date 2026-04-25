#include<iostream>
#include<string>
#include<set>
#include<fstream>
#include<vector>
#include<thread>
#include<chrono>
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
                        "\'https://tdx.transportdata.tw/api/basic/v2/Bus/Route/City/" +city[i] +"?$select=RouteUID,RouteID,RouteName,BusRouteType,DepartureStopNameZh,DestinationStopNameZh&$format=JSON\' > temp.json";
        system(s.c_str());
        std::set<std::string> s;
        std::ifstream f("temp.json");
        if (f.good()) {
            nlohmann::json json = nlohmann::json::parse(f);
            if (json.is_array()) {
                for (auto &j : json) {
                    nlohmann::json temp;
                    temp["RouteUID"] = j["RouteUID"];
                    temp["RouteName"] = j["RouteName"]["Zh_tw"];
                    temp["City"] = city[i];
                    temp["Type"] = j["BusRouteType"];
                    temp["DepartureStopNameZh"] = j["DepartureStopNameZh"];
                    temp["DestinationStopNameZh"] = j["DestinationStopNameZh"];
                    if(j.contains("HasSubRoutes")){
                        nlohmann::json temp["SubRoutes"] = nlohmann::json::array();
                        for (auto &k : j["SubRoutes"]) {
                            if(s.count(k["SubRouteUID"].get<std::string>())) continue;
                            nlohmann::json temp2;
                            temp2["SubRouteUID"] = k["SubRouteUID"];
                            temp2["SubRouteName"] = k["SubRouteName"]["Zh_tw"];
                            if(k.contains("DepartureStopNameZh")) temp2["DepartureStopNameZh"] = k["DepartureStopNameZh"];
                            if(k.contains("DestinationStopNameZh")) temp2["DestinationStopNameZh"] = k["DestinationStopNameZh"];
                            temp["SubRoutes"].emplace_back(temp2);
                            s.emplace(k["SubRouteUID"].get<std::string>());
                        }
                    }
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
                temp["City"] = "InterCity";
                temp["Type"] = j["BusRouteType"];
                temp["DepartureStopNameZh"] = j["DepartureStopNameZh"];
                temp["DestinationStopNameZh"] = j["DestinationStopNameZh"];
                if(j.contains("HasSubRoutes")){
                    nlohmann::json temp["SubRoutes"] = nlohmann::json::array();
                        for (auto &k : j["SubRoutes"]) {
                        if(s.count(k["SubRouteUID"].get<std::string>())) continue;
                        nlohmann::json temp2;
                        temp2["SubRouteUID"] = k["SubRouteUID"];
                        temp2["SubRouteName"] = k["SubRouteName"]["Zh_tw"];
                        if(k.contains("DepartureStopNameZh")) temp2["DepartureStopNameZh"] = k["DepartureStopNameZh"];
                        if(k.contains("DestinationStopNameZh")) temp2["DestinationStopNameZh"] = k["DestinationStopNameZh"];
                        temp["SubRoutes"].emplace_back(temp2);
                        s.emplace(k["SubRouteUID"].get<std::string>());
                    }
                }
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
    }
    catch (std::exception &e) {
        std::cerr << "error: " << e.what() << '\n';
    }
    return 0;
}