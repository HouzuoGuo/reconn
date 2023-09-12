import { Component } from '@angular/core';
import { ReadbackService, MyRequest } from './readback.service';
import { Observable } from 'rxjs';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  readbackResp: Observable<MyRequest>;
  constructor(private readback: ReadbackService) {
    this.readbackResp = readback.readback();
  }
}
