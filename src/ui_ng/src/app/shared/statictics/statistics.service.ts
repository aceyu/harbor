// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import { Injectable } from '@angular/core';
import {Http, URLSearchParams} from '@angular/http';
import 'rxjs/add/operator/toPromise';

import { Statistics } from './statistics';
import { Volumes } from './volumes';
import {buildHttpRequestOptions, HTTP_GET_OPTIONS} from "../shared.utils";

const statisticsEndpoint = "/api/statistics";
const volumesEndpoint = "/api/systeminfo/volumes";
/**
 * Declare service to handle the top repositories
 *
 *
 * @export
 * @class GlobalSearchService
 */
@Injectable()
export class StatisticsService {

    constructor(private http: Http) { }

    getStatistics(isRemote?: boolean): Promise<Statistics> {
        if (isRemote) {
            let params = new URLSearchParams();
            params.set('isRemote', 'true');
            let option = buildHttpRequestOptions(params)
            option.search = params
            return this.http.get(statisticsEndpoint, option).toPromise()
                .then(response => response.json() as Statistics)
                .catch(error => Promise.reject(error));
        }
        return this.http.get(statisticsEndpoint, HTTP_GET_OPTIONS).toPromise()
        .then(response => response.json() as Statistics)
        .catch(error => Promise.reject(error));
    }

    getVolumes(): Promise<Volumes> {
        return this.http.get(volumesEndpoint, HTTP_GET_OPTIONS).toPromise()
        .then(response => response.json() as Volumes)
        .catch(error => Promise.reject(error));
    }
}
