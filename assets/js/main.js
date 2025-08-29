$(function () {
    /*=======================
                UI Slider Range JS
    =========================*/

    // Simpan lokasi saat tombol diklik
       document.getElementById("confirm-location").addEventListener("click", function () {
        const lat = marker.getPosition().lat();
        const lng = marker.getPosition().lng();
        let cityID = $(".city_id").val()
        let courier = $(".courier").val()
        requestPrice(cityID, "instant", courier,lat, lng)
        alert(`Latitude: ${lat}, Longitude: ${lng} telah dipilih.`);
    });


    $("#slider-range").slider({
        range: true,
        min: 0,
        max: 2500,
        values: [10, 2500],
        slide: function (event, ui) {
            $("#amount").val("$" + ui.values[0] + " - $" + ui.values[1]);
        }
    });
    $("#amount").val("$" + $("#slider-range").slider("values", 0) +
        " - $" + $("#slider-range").slider("values", 1));

    let domShippingCalculationMsg = $("#shipping-calculation-msg")

    $(".province_id").change(function () {
        provinceID = $(".province_id").val()

        $(".city_id").find("option")
            .remove()
            .end()
            .append('<option value="">Pilih Kota / Kabupaten</option>')

        $.ajax({
            url: "/carts/cities?province_id=" + provinceID,
            method: "GET",
            success: function (result) {
                $.each(result.data, function (i, city) {
                    console.log(city.city_id + city.city_name);
                    $(".city_id").append(`<option value="${city.city_id}">${city.city_name}</option>`)
                });
            }
        })
    });

    // Event handler untuk city selection (hanya untuk regular delivery)
    $(".city_id").change(function () {
        let cityID = $(this).val();
        let courierValue = $(".courier").val();
        
        // Hanya update jika bukan instant delivery
        if (cityID && !["grab", "gojek", "deliveree", "lalamove"].includes(courierValue)) {
            let cityText = $(this).find("option:selected").text();
            domShippingCalculationMsg.html(`<div class="alert alert-info small">Tujuan Regular: ${cityText}</div>`);
        } else if (!["grab", "gojek", "deliveree", "lalamove"].includes(courierValue)) {
            domShippingCalculationMsg.html('');
        }
        
        // Reset shipping options untuk regular delivery
        if (!["grab", "gojek", "deliveree", "lalamove"].includes(courierValue)) {
            $(".shipping_fee_options").empty().append('<option value="">Pilih Paket</option>');
        }
    });

    $(".courier").change(function () {
        let cityID = $(".city_id").val()
        let courier = $(".courier").val()
        let latitude = $("#latitude").val()
        let longitude = $("#longitude").val()

        $(".shipping_fee_options").find("option")
            .remove()
            .end()
            .append('<option value="">Pilih Paket</option>')

            if (["grab", "gojek", "deliveree", "lalamove"].includes(courier)) {
                // Clear previous area info, akan di-update oleh Google Maps
                domShippingCalculationMsg.html('<div class="alert alert-warning small">Pilih lokasi di peta yang muncul untuk instant delivery</div>');
                requestPrice(cityID, "instant", courier, latitude, longitude)
            }else if(courier == 'pickup'){
                // Handle pickup option
                domShippingCalculationMsg.html('<div class="alert alert-success small"><strong>Pickup di Toko:</strong><br/>Toko Shafirda, Samarinda<br/>Gratis ongkos kirim</div>');
                $(".shipping_fee_options").empty();
                $(".shipping_fee_options").append(`<option value="pickup" selected>Pickup - Gratis (Ambil di Toko)</option>`);
            }
            else{
                // Update area info dari dropdown untuk regular delivery
                let cityText = $(".city_id").find("option:selected").text();
                domShippingCalculationMsg.html(`<div class="alert alert-info small">Tujuan Regular: ${cityText}</div>`);
                requestPrice(cityID, "regular", courier, latitude, longitude)
            }
      
    });

    const requestPrice = (cityID, type, courier, latitude, longitude) =>{
        // DEBUG: Log what we're sending
        console.log("🔍 SENDING TO SHIPPING CALCULATION:");
        console.log("   city_id:", cityID);
        console.log("   courier:", courier);
        console.log("   cour_type:", type);
        console.log("   latitude:", latitude);
        console.log("   longitude:", longitude);
        
        $.ajax({
            url: "/carts/calculate-shipping",
            method: "POST",
            data: {
                city_id: cityID,
                courier: courier,
                cour_type: type,
                latitude: latitude,
                longitude: longitude,
                
            },
            success: function (result) {
                domShippingCalculationMsg.html('');
                $(".shipping_fee_options").empty()
                
                // Tampilkan informasi area jika tersedia
                if (result.data.origin && result.data.destination) {
                    let areaInfo = '<div class="alert alert-info small">';
                    areaInfo += '<strong>Rute Pengiriman:</strong><br/>';
                    areaInfo += `Dari: ${result.data.origin.area_name}<br/>`;
                    areaInfo += `Ke: ${result.data.destination.area_name}`;
                    areaInfo += '</div>';
                    domShippingCalculationMsg.html(areaInfo);
                }
                
                // Populate shipping options
                const shippingOptions = result.data.pricing || result.data;
                $.each(shippingOptions, function (i, shipping_fee_option) {
                    let optionText = `${shipping_fee_option.duration} - Rp ${shipping_fee_option.price.toLocaleString('id-ID')}`;
                    if (shipping_fee_option.courier_service_name) {
                        optionText += ` (${shipping_fee_option.courier_service_name})`;
                    }
                    $(".shipping_fee_options").append(`<option value="${shipping_fee_option.courier_name}">${optionText}</option>`);
                });
            },
            error: function (xhr, status, error) {
                console.error("❌ SHIPPING CALCULATION ERROR:");
                console.error("   Status:", status);
                console.error("   Error:", error);
                console.error("   Response:", xhr.responseText);
                domShippingCalculationMsg.html(`<div class="alert alert-warning">Perhitungan ongkos kirim gagal! Error: ${error}</div>`);
            }
        })
    }

    const applyPrice = (cityID, type, courier, shippingFee, latitude, longitude) =>{
        let postData = {
            shipping_package: shippingFee.split("-")[0].trim(),
            city_id: cityID,
            courier: courier,
            cour_type: type
        };
        
        // Tambahkan koordinat untuk instant delivery
        if (type === "instant" && latitude && longitude) {
            postData.latitude = latitude;
            postData.longitude = longitude;
        }
        
        $.ajax({
            url: "/carts/apply-shipping",
            method: "POST",
            data: postData,
            success: function (result) {
                if (result.data.grand_total) {
                    $("#grand-total").text(`Rp ${parseFloat(result.data.grand_total).toLocaleString('id-ID')}`);
                    
                    // Update shipping info jika ada
                    if (result.data.origin && result.data.destination) {
                        let shippingInfo = '<div class="alert alert-success small">';
                        shippingInfo += '<strong>Paket Dipilih:</strong><br/>';
                        shippingInfo += `${result.data.origin.area_name} ke ${result.data.destination.area_name}<br/>`;
                        if (result.data.courier_info) {
                            shippingInfo += `Kurir: ${result.data.courier_info.courier_name}`;
                            if (result.data.courier_info.service_name) {
                                shippingInfo += ` (${result.data.courier_info.service_name})`;
                            }
                            if (result.data.courier_info.duration) {
                                shippingInfo += `<br/>Estimasi: ${result.data.courier_info.duration}`;
                            }
                        }
                        shippingInfo += `<br/>Ongkir: Rp ${parseFloat(result.data.shipping_fee).toLocaleString('id-ID')}`;
                        shippingInfo += '</div>';
                        domShippingCalculationMsg.html(shippingInfo);
                    }
                }
            },
            error: function (e) {
                domShippingCalculationMsg.html(`<div class="alert alert-warning">Pemilihan paket ongkir gagal!</div>`);
            }
        })
    }

    $(".shipping_fee_options").change(function () {
        let cityID = $(".city_id").val()
        let courier = $(".courier").val()
        let shippingFee = $(this).val();
        let latitude = $("#latitude").val()
        let longitude = $("#longitude").val()

        if (["grab", "gojek", "deliveree", "lalamove"].includes(courier)) {
            applyPrice(cityID, "instant", courier, shippingFee, latitude, longitude)
        }else if(courier == 'pickup'){
            // Pickup tidak perlu apply shipping cost
            $("#grand-total").text("Total akan dihitung saat checkout");
        }else{
            applyPrice(cityID, "regular", courier, shippingFee, latitude, longitude)
        }

     
    });

    // Form validation sebelum submit
    $("#calculate-shipping").on("submit", function(e) {
        let courier = $(".courier").val();
        let shippingFee = $(".shipping_fee_options").val();
        let firstName = $("#first_name").val();
        let lastName = $("#last_name").val();
        let address = $("#address1").val();
        let phone = $("#phone").val();
        
        // Validasi basic
        if (!courier) {
            alert("Pilih kurir terlebih dahulu");
            e.preventDefault();
            return false;
        }
        
        if (!shippingFee && courier !== 'pickup') {
            alert("Pilih paket pengiriman terlebih dahulu");
            e.preventDefault();
            return false;
        }
        
        if (!firstName || !lastName || !address || !phone) {
            alert("Lengkapi detail pengiriman (nama, alamat, telepon)");
            e.preventDefault();
            return false;
        }
        
        // Validasi khusus untuk instant delivery
        if (["grab", "gojek", "deliveree", "lalamove"].includes(courier)) {
            let lat = $("#latitude").val();
            let lng = $("#longitude").val();
            
            if (!lat || !lng) {
                alert("Pilih lokasi di peta untuk instant delivery");
                e.preventDefault();
                return false;
            }
        }
        
        return true;
    });
    
});
