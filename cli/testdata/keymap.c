#include QMK_KEYBOARD_H
#include "version.h"
#define MOON_LED_LEVEL LED_LEVEL
#ifndef ZSA_SAFE_RANGE
#define ZSA_SAFE_RANGE SAFE_RANGE
#endif

enum custom_keycodes {
  RGB_SLD = ZSA_SAFE_RANGE,
};



#define DUAL_FUNC_0 LT(3, KC_F5)

const uint16_t PROGMEM keymaps[][MATRIX_ROWS][MATRIX_COLS] = {
  [0] = LAYOUT_moonlander(
    KC_ESCAPE,      KC_1,           KC_2,           KC_3,           KC_4,           KC_5,           KC_F11,                                         KC_TRANSPARENT, KC_6,           KC_7,           KC_8,           KC_9,           KC_0,           KC_EQUAL,       
    KC_TRANSPARENT, KC_Q,           KC_W,           KC_E,           KC_R,           KC_T,           KC_TRANSPARENT,                                 KC_TRANSPARENT, KC_Y,           KC_U,           KC_I,           KC_O,           KC_P,           KC_MINUS,       
    KC_TRANSPARENT, MT(MOD_LGUI, KC_A),MT(MOD_LCTL, KC_S),MT(MOD_LALT, KC_D),MT(MOD_LSFT, KC_F),KC_G,           KC_TAB,                                                                         KC_BSPC,        KC_H,           MT(MOD_RSFT, KC_J),MT(MOD_RALT, KC_K),MT(MOD_RCTL, KC_L),DUAL_FUNC_0,    KC_QUOTE,       
    KC_TRANSPARENT, KC_Z,           KC_X,           KC_C,           KC_V,           KC_B,                                           KC_N,           KC_M,           KC_COMMA,       KC_DOT,         KC_SLASH,       KC_BSLS,        
    KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT,                                                                                                 KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_AUDIO_VOL_DOWN,KC_AUDIO_VOL_UP,KC_AUDIO_MUTE,  
    OSL(1),         OSL(2),         TG(2),                          KC_TRANSPARENT, KC_ENTER,       KC_SPACE
  ),
  [1] = LAYOUT_moonlander(
    KC_TRANSPARENT, KC_EXLM,        KC_AT,          KC_HASH,        KC_DLR,         KC_PERC,        KC_TRANSPARENT,                                 KC_TRANSPARENT, KC_CIRC,        KC_AMPR,        KC_ASTR,        KC_UNDS,        KC_PIPE,        KC_TRANSPARENT, 
    KC_TRANSPARENT, KC_TRANSPARENT, LGUI(KC_X),     LGUI(KC_C),     LGUI(KC_V),     KC_PIPE,        KC_TRANSPARENT,                                 KC_TRANSPARENT, KC_LCBR,        KC_RCBR,        KC_LPRN,        KC_RPRN,        KC_GRAVE,       KC_PLUS,        
    KC_TRANSPARENT, KC_TRANSPARENT, LGUI(KC_SPACE), LGUI(KC_S),     KC_BSPC,        KC_GRAVE,       KC_TRANSPARENT,                                                                 KC_TRANSPARENT, KC_LEFT,        KC_DOWN,        KC_UP,          KC_RIGHT,       KC_SCLN,        KC_DQUO,        
    KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_SPACE,       KC_ENTER,                                       KC_LABK,        KC_RABK,        KC_LBRC,        KC_RBRC,        KC_TILD,        KC_TRANSPARENT, 
    KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT,                                                                                                 KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, 
    KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT,                 KC_TRANSPARENT, KC_TRANSPARENT, KC_TAB
  ),
  [2] = LAYOUT_moonlander(
    KC_ESCAPE,      KC_1,           KC_2,           KC_3,           KC_4,           KC_5,           KC_TRANSPARENT,                                 KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, 
    KC_GRAVE,       KC_TRANSPARENT, KC_Q,           KC_W,           KC_E,           KC_R,           KC_TRANSPARENT,                                 KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, 
    KC_TRANSPARENT, KC_LEFT_SHIFT,  KC_A,           KC_S,           KC_D,           KC_F,           KC_TRANSPARENT,                                                                 KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, 
    KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT,                                 KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, 
    KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_LEFT_ALT,    KC_LEFT_CTRL,   KC_TRANSPARENT,                                                                                                 KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT, 
    KC_SPACE,       KC_ENTER,       KC_TRANSPARENT,                 KC_TRANSPARENT, KC_TRANSPARENT, KC_TRANSPARENT
  ),
};

const char chordal_hold_layout[MATRIX_ROWS][MATRIX_COLS] PROGMEM = LAYOUT(
  'L', 'L', 'L', 'L', 'L', 'L', 'L', 'R', 'R', 'R', 'R', 'R', 'R', 'R',
  'L', 'L', 'L', 'L', 'L', 'L', 'L', 'R', 'R', 'R', 'R', 'R', 'R', 'R',
  'L', 'L', 'L', 'L', 'L', 'L', 'L', 'R', 'R', 'R', 'R', 'R', 'R', 'R',
  'L', 'L', 'L', 'L', 'L', 'L', 'R', 'R', 'R', 'R', 'R', 'R',
  'L', 'L', 'L', 'L', 'L', '*', '*', 'R', 'R', 'R', 'R', 'R',
                 '*', '*', '*', '*', '*', '*'
);




extern rgb_config_t rgb_matrix_config;

RGB hsv_to_rgb_with_value(HSV hsv) {
  RGB rgb = hsv_to_rgb( hsv );
  float f = (float)rgb_matrix_config.hsv.v / UINT8_MAX;
  return (RGB){ f * rgb.r, f * rgb.g, f * rgb.b };
}

void keyboard_post_init_user(void) {
  rgb_matrix_enable();
}

const uint8_t PROGMEM ledmap[][RGB_MATRIX_LED_COUNT][3] = {
    [0] = { {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {234,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {234,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {234,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {234,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {234,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {234,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {234,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {234,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255} },

    [1] = { {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {41,255,255}, {41,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {41,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {0,245,245}, {0,245,245}, {0,245,245}, {219,255,255}, {0,245,245}, {0,245,245}, {0,245,245}, {0,245,245}, {219,255,255}, {131,255,255}, {41,255,255}, {74,255,255}, {41,255,255}, {219,255,255}, {131,255,255}, {41,255,255}, {74,255,255}, {41,255,255}, {219,255,255}, {131,255,255}, {41,255,255}, {74,255,255}, {41,255,255}, {219,255,255}, {131,255,255}, {41,255,255}, {74,255,255}, {41,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255}, {219,255,255} },

    [2] = { {131,255,255}, {131,255,255}, {161,244,195}, {151,255,171}, {151,255,171}, {131,255,255}, {131,255,255}, {131,255,255}, {161,244,195}, {151,255,171}, {131,255,255}, {131,255,255}, {131,255,255}, {151,255,171}, {151,255,171}, {131,255,255}, {131,255,255}, {131,255,255}, {151,255,171}, {131,255,255}, {131,255,255}, {131,255,255}, {131,255,255}, {151,255,171}, {131,255,255}, {131,255,255}, {131,255,255}, {151,255,171}, {151,255,171}, {151,255,171}, {151,255,171}, {151,255,171}, {131,255,255}, {131,255,255}, {151,255,171}, {151,255,171}, {152,255,255}, {151,255,171}, {151,255,171}, {151,255,171}, {151,255,171}, {152,255,255}, {151,255,171}, {151,255,171}, {151,255,171}, {151,255,171}, {152,255,255}, {152,255,255}, {151,255,171}, {151,255,171}, {151,255,171}, {152,255,255}, {152,255,255}, {152,255,255}, {151,255,171}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {152,255,255}, {151,255,171}, {151,255,171}, {151,255,171}, {151,255,171} },

};

void set_layer_color(int layer) {
  for (int i = 0; i < RGB_MATRIX_LED_COUNT; i++) {
    HSV hsv = {
      .h = pgm_read_byte(&ledmap[layer][i][0]),
      .s = pgm_read_byte(&ledmap[layer][i][1]),
      .v = pgm_read_byte(&ledmap[layer][i][2]),
    };
    if (!hsv.h && !hsv.s && !hsv.v) {
        rgb_matrix_set_color( i, 0, 0, 0 );
    } else {
        RGB rgb = hsv_to_rgb_with_value(hsv);
        rgb_matrix_set_color(i, rgb.r, rgb.g, rgb.b);
    }
  }
}

bool rgb_matrix_indicators_user(void) {
  if (rawhid_state.rgb_control) {
      return false;
  }
  if (!keyboard_config.disable_layer_led) { 
    switch (biton32(layer_state)) {
      case 0:
        set_layer_color(0);
        break;
      case 1:
        set_layer_color(1);
        break;
      case 2:
        set_layer_color(2);
        break;
     default:
        if (rgb_matrix_get_flags() == LED_FLAG_NONE) {
          rgb_matrix_set_color_all(0, 0, 0);
        }
    }
  } else {
    if (rgb_matrix_get_flags() == LED_FLAG_NONE) {
      rgb_matrix_set_color_all(0, 0, 0);
    }
  }

  return true;
}




bool process_record_user(uint16_t keycode, keyrecord_t *record) {
  switch (keycode) {
  case QK_MODS ... QK_MODS_MAX:
    // Mouse and consumer keys (volume, media) with modifiers work inconsistently across operating systems,
    // this makes sure that modifiers are always applied to the key that was pressed.
    if (IS_CONSUMER_KEYCODE(QK_MODS_GET_BASIC_KEYCODE(keycode))) {
      if (record->event.pressed) {
        add_mods(QK_MODS_GET_MODS(keycode));
        send_keyboard_report();
        wait_ms(2);
        register_code(QK_MODS_GET_BASIC_KEYCODE(keycode));
        return false;
      } else {
        wait_ms(2);
        del_mods(QK_MODS_GET_MODS(keycode));
      }
    }
    break;

    case DUAL_FUNC_0:
      if (record->tap.count > 0) {
        if (record->event.pressed) {
          register_code16(KC_COLN);
        } else {
          unregister_code16(KC_COLN);
        }
      } else {
        if (record->event.pressed) {
          register_code16(KC_RIGHT_GUI);
        } else {
          unregister_code16(KC_RIGHT_GUI);
        }  
      }  
      return false;
    case RGB_SLD:
        if (rawhid_state.rgb_control) {
            return false;
        }
        if (record->event.pressed) {
            rgblight_mode(1);
        }
        return false;
  }
  return true;
}

